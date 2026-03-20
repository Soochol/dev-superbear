#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────
# PR Docker Orchestration Script (범용)
# ─────────────────────────────────────────────
# 프로젝트의 docker-compose 파일을 격리된 project name으로 실행.
# 서비스 구성, 포트, 환경변수는 compose 파일에 정의된 대로 사용.
#
# Usage:
#   pr-docker.sh up [--port-offset N]   # 서비스 기동
#   pr-docker.sh down                   # 이 worktree만 정리
#   pr-docker.sh down-all               # 전체 정리
#   pr-docker.sh status                 # 현재 상태 확인
# ─────────────────────────────────────────────

# ─── Colors ───
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}[pr-docker]${NC} $1"; }
ok()   { echo -e "${GREEN}[pr-docker]${NC} $1"; }
warn() { echo -e "${YELLOW}[pr-docker]${NC} $1"; }
err()  { echo -e "${RED}[pr-docker]${NC} $1" >&2; }

# ─── Detect project root ───
find_project_root() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        if [[ -f "$dir/docker-compose.test.yml" ]] || \
           [[ -f "$dir/docker-compose.yml" ]] || \
           [[ -f "$dir/compose.yml" ]] || \
           [[ -f "$dir/compose.yaml" ]]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    err "Could not find project root (no compose file found)"
    exit 1
}

PROJECT_ROOT="$(find_project_root)"

# ─── Detect compose file ───
find_compose_file() {
    # Prefer test-specific compose, then standard names
    for f in "docker-compose.test.yml" "docker-compose.yml" "compose.yml" "compose.yaml"; do
        if [[ -f "$PROJECT_ROOT/$f" ]]; then
            echo "$PROJECT_ROOT/$f"
            return 0
        fi
    done
    err "No compose file found"
    exit 1
}

COMPOSE_FILE="$(find_compose_file)"

# ─── Detect worktree identity ───
detect_identity() {
    local wt_dir
    wt_dir="$(git -C "$PROJECT_ROOT" rev-parse --show-toplevel 2>/dev/null || echo "$PROJECT_ROOT")"
    local dir_name
    dir_name="$(basename "$wt_dir")"
    echo "$dir_name" | sed 's/[^a-zA-Z0-9_-]/-/g' | cut -c1-50
}

detect_branch() {
    git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown"
}

IDENTITY="$(detect_identity)"
BRANCH="$(detect_branch)"

# ─── Project name for Docker Compose ───
# This is the key to isolation: each worktree gets a unique project name,
# so containers, networks, and volumes are fully separated.
PROJECT_NAME="pr-${IDENTITY}"

# ─── Port offset ───
compute_offset() {
    local main_dir
    main_dir="$(git -C "$PROJECT_ROOT" worktree list --porcelain 2>/dev/null | head -1 | awk '{print $2}' || echo "")"
    main_dir="$(basename "$main_dir" 2>/dev/null || echo "")"
    if [[ "$IDENTITY" == "$main_dir" ]] || [[ -z "$main_dir" ]]; then
        echo 0
        return
    fi
    local hash
    hash=$(echo -n "$IDENTITY" | cksum | awk '{print $1}')
    echo $(( hash % 100 ))
}

# Parse CLI args
PORT_OFFSET=""
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --port-offset)
                if [[ -z "${2:-}" ]]; then
                    err "--port-offset requires a numeric value"
                    exit 1
                fi
                PORT_OFFSET="$2"
                shift 2
                ;;
            *)
                err "Unknown argument: $1"
                exit 1
                ;;
        esac
    done
}

# ─── Docker Compose wrapper ───
dc() {
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" "$@"
}

# ═══════════════════════════════════════
# Commands
# ═══════════════════════════════════════

cmd_up() {
    log "Starting PR environment: $IDENTITY (branch: $BRANCH)"
    log "  Compose file:  $(basename "$COMPOSE_FILE")"
    log "  Project name:  $PROJECT_NAME"
    log "  Port offset:   $PORT_OFFSET"
    echo ""

    # Export port variables so compose file can reference them
    export PG_HOST_PORT=$((5433 + PORT_OFFSET))
    export API_HOST_PORT=$((8080 + PORT_OFFSET))

    if [[ "$PORT_OFFSET" -ne 0 ]]; then
        log "Applying port offset: $PORT_OFFSET (PG=$PG_HOST_PORT, API=$API_HOST_PORT)"
    fi

    dc up -d --build --wait 2>&1 | while read -r line; do log "  $line"; done

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo -e "${CYAN}  PR Test Environment Ready${NC}"
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo ""
    echo -e "  Identity:     ${GREEN}${IDENTITY}${NC} (branch: ${BRANCH})"
    echo -e "  Project:      ${GREEN}${PROJECT_NAME}${NC}"
    echo -e "  Compose file: ${GREEN}$(basename "$COMPOSE_FILE")${NC}"
    echo ""

    # Show running services and their ports
    echo -e "  ${CYAN}Services:${NC}"
    dc ps --format "    {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || true
    echo ""
    echo -e "  Stop: ${CYAN}bash $0 down${NC}"
    echo ""
}

cmd_down() {
    log "Stopping PR environment: $IDENTITY (branch: $BRANCH)"

    dc down -v --remove-orphans 2>&1 | while read -r line; do log "  $line"; done

    ok "Cleanup complete: $IDENTITY"
}

cmd_down_all() {
    log "Stopping ALL PR test environments..."

    # Find all pr-* compose projects (no python3 dependency)
    local projects
    projects=$(docker compose ls --format json 2>/dev/null | \
        jq -r '.[] | select(.Name | startswith("pr-")) | .Name + "|" + .ConfigFiles' 2>/dev/null || \
        docker compose ls 2>/dev/null | awk '/^pr-/ {print $1 "|" $3}' || true)

    if [[ -n "$projects" ]]; then
        while IFS='|' read -r name config; do
            if [[ -n "$config" ]]; then
                log "Stopping project: $name"
                docker compose -f "$config" -p "$name" down -v --remove-orphans 2>/dev/null || true
                ok "  Removed: $name"
            fi
        done <<< "$projects"
    fi

    # Fallback: stop any remaining pr-* containers
    local containers
    containers=$(docker ps -aq --filter "name=^pr-" 2>/dev/null || true)
    if [[ -n "$containers" ]]; then
        echo "$containers" | xargs docker stop 2>/dev/null || true
        echo "$containers" | xargs docker rm 2>/dev/null || true
        ok "Remaining PR containers removed"
    fi

    # Remove pr-* networks
    local networks
    networks=$(docker network ls --filter "name=^pr-" --format "{{.Name}}" 2>/dev/null || true)
    if [[ -n "$networks" ]]; then
        echo "$networks" | xargs docker network rm 2>/dev/null || true
        ok "PR networks removed"
    fi

    ok "Full cleanup complete"
}

cmd_status() {
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo -e "${CYAN}  PR Test Environment Status${NC}"
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo ""

    echo -e "  ${CYAN}Identity:${NC}       $IDENTITY"
    echo -e "  ${CYAN}Branch:${NC}         $BRANCH"
    echo -e "  ${CYAN}Project:${NC}        $PROJECT_NAME"
    echo -e "  ${CYAN}Compose file:${NC}   $(basename "$COMPOSE_FILE")"
    echo ""

    # Current project services
    echo -e "  ${CYAN}This worktree:${NC}"
    local services
    services=$(dc ps --format "    {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || true)
    if [[ -n "$services" ]]; then
        echo "$services"
    else
        echo "    (no services running)"
    fi

    # All PR projects
    echo ""
    echo -e "  ${CYAN}All PR containers:${NC}"
    local all_containers
    all_containers=$(docker ps --filter "name=^pr-" --format "    {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || true)
    if [[ -n "$all_containers" ]]; then
        echo "$all_containers"
    else
        echo "    (none)"
    fi

    echo ""
}

# ═══════════════════════════════════════
# Main
# ═══════════════════════════════════════

COMMAND="${1:-status}"
shift || true

parse_args "$@"

if [[ -z "$PORT_OFFSET" ]]; then
    PORT_OFFSET="$(compute_offset)"
fi

case "$COMMAND" in
    up)       cmd_up ;;
    down)     cmd_down ;;
    down-all) cmd_down_all ;;
    status)   cmd_status ;;
    *)
        echo "Usage: $0 {up|down|down-all|status} [--port-offset N]"
        echo ""
        echo "Commands:"
        echo "  up         Start services from compose file (isolated project)"
        echo "  down       Stop and remove this worktree's services"
        echo "  down-all   Stop ALL PR test environments"
        echo "  status     Show current state"
        echo ""
        echo "Options:"
        echo "  --port-offset N   Manual port offset (default: auto from worktree name)"
        exit 1
        ;;
esac
