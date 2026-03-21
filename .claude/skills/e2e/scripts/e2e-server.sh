#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}/../templates"
PROJECT_DIR="$(git rev-parse --show-toplevel)"

# --- worktree 감지 ---
detect_worktree() {
  local git_dir
  git_dir="$(git rev-parse --git-dir)"
  if [[ "$git_dir" == *"/worktrees/"* ]]; then
    basename "$git_dir"
  else
    echo "main"
  fi
}

# --- 포트 할당 ---
allocate_ports() {
  local name="$1"
  local offset=0

  if [[ "$name" != "main" ]]; then
    offset=$(( $(echo -n "$name" | cksum | awk '{print $1}') % 9 + 1 ))
  fi

  local max_attempts=20
  while (( offset < max_attempts )); do
    local port_root=$(( 3100 + offset ))
    local port_api=$(( 3300 + offset ))
    local port_worker=$(( 3400 + offset ))

    if ! ss -tlnp 2>/dev/null | grep -qE ":($port_root|$port_api|$port_worker) " ; then
      echo "${port_root} ${port_api} ${port_worker}"
      return 0
    fi
    offset=$(( offset + 1 ))
  done

  echo "ERROR: No available ports found after $max_attempts attempts" >&2
  return 1
}

# --- .env.e2e 생성 ---
write_env() {
  local worktree_name="$1"
  local port_root="$2"
  local port_api="$3"
  local port_worker="$4"
  local env_file="${PROJECT_DIR}/.env.e2e"

  cat > "$env_file" <<EOF
E2E_PORT_ROOT=${port_root}
E2E_PORT_API=${port_api}
E2E_PORT_WORKER=${port_worker}
WORKTREE_NAME=${worktree_name}
EOF
  echo "Generated ${env_file}"
}

# --- docker compose 헬퍼 ---
compose_infra() {
  docker compose \
    --project-directory "${PROJECT_DIR}" \
    -f "${TEMPLATE_DIR}/docker-compose.infra.yml" \
    -p superbear-infra \
    "$@"
}

compose_worktree() {
  local worktree_name="$1"
  shift
  docker compose \
    --project-directory "${PROJECT_DIR}" \
    -f "${TEMPLATE_DIR}/docker-compose.test.yml" \
    -p "superbear-e2e-${worktree_name}" \
    --env-file "${PROJECT_DIR}/.env.e2e" \
    "$@"
}

# --- health check ---
wait_for_healthy() {
  local url="$1"
  local label="$2"
  local timeout=60
  local elapsed=0

  echo -n "Waiting for ${label}..."
  while (( elapsed < timeout )); do
    if curl -sf "$url" > /dev/null 2>&1; then
      echo " ready"
      return 0
    fi
    sleep 2
    elapsed=$(( elapsed + 2 ))
    echo -n "."
  done

  echo " TIMEOUT"
  return 1
}

# --- 명령어 ---
cmd_up() {
  local worktree_name
  worktree_name="$(detect_worktree)"
  echo "Worktree: ${worktree_name}"

  local env_file="${PROJECT_DIR}/.env.e2e"
  if [[ -f "$env_file" ]]; then
    echo "Reusing existing .env.e2e"
    source "$env_file"
  else
    local ports
    ports="$(allocate_ports "$worktree_name")"
    read -r port_root port_api port_worker <<< "$ports"
    write_env "$worktree_name" "$port_root" "$port_api" "$port_worker"
    E2E_PORT_ROOT="$port_root"
    E2E_PORT_API="$port_api"
    E2E_PORT_WORKER="$port_worker"
  fi

  # 1. 공용 인프라 (이미 떠있으면 skip)
  if ! docker compose -p superbear-infra ps --status running 2>/dev/null | grep -q "postgres"; then
    echo "Starting shared infra..."
    compose_infra up -d
    compose_infra exec postgres sh -c 'until pg_isready -U nexus -d nexus_test; do sleep 1; done'
    echo "Shared infra ready"
  else
    echo "Shared infra already running"
  fi

  # 2. worktree별 서비스
  echo "Starting worktree services (ports: root=${E2E_PORT_ROOT}, api=${E2E_PORT_API}, worker=${E2E_PORT_WORKER})..."
  compose_worktree "$worktree_name" up -d --build

  # 3. health check
  wait_for_healthy "http://localhost:${E2E_PORT_API}/api/v1/health" "API" || {
    echo "Health check failed, tearing down..."
    cmd_down
    exit 1
  }
  wait_for_healthy "http://localhost:${E2E_PORT_ROOT}" "Root App" || {
    echo "Health check failed, tearing down..."
    cmd_down
    exit 1
  }

  echo ""
  echo "=== E2E Environment Ready ==="
  echo "Root App:  http://localhost:${E2E_PORT_ROOT}"
  echo "API:       http://localhost:${E2E_PORT_API}"
  echo "Worker:    http://localhost:${E2E_PORT_WORKER}"
  echo "============================="
}

cmd_down() {
  local worktree_name
  worktree_name="$(detect_worktree)"
  local all=false

  if [[ "${1:-}" == "--all" ]]; then
    all=true
  fi

  echo "Stopping worktree services (${worktree_name})..."
  compose_worktree "$worktree_name" down --remove-orphans 2>/dev/null || true

  if $all; then
    echo "Stopping shared infra..."
    compose_infra down --remove-orphans 2>/dev/null || true
  fi

  rm -f "${PROJECT_DIR}/.env.e2e"
  echo "Done"
}

cmd_status() {
  echo "=== Shared Infra ==="
  compose_infra ps 2>/dev/null || echo "Not running"
  echo ""
  echo "=== Worktree Services ($(detect_worktree)) ==="
  local worktree_name
  worktree_name="$(detect_worktree)"
  compose_worktree "$worktree_name" ps 2>/dev/null || echo "Not running"

  if [[ -f "${PROJECT_DIR}/.env.e2e" ]]; then
    echo ""
    echo "=== Ports ==="
    cat "${PROJECT_DIR}/.env.e2e"
  fi
}

# --- main ---
case "${1:-}" in
  up)     cmd_up ;;
  down)   cmd_down "${2:-}" ;;
  status) cmd_status ;;
  *)
    echo "Usage: e2e-server.sh {up|down [--all]|status}"
    exit 1
    ;;
esac
