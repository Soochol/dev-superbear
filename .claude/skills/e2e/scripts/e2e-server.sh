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
    offset=$(( $(echo -n "$name" | cksum | awk '{print $1}') % 100 + 1 ))
  fi

  local max_attempts=50
  local attempt=0
  while (( attempt < max_attempts )); do
    local port_root=$(( 10000 + offset + attempt ))
    local port_api=$(( 10100 + offset + attempt ))
    local port_worker=$(( 10200 + offset + attempt ))

    if ! ss -tlnp 2>/dev/null | grep -qE ":($port_root|$port_api|$port_worker) " ; then
      echo "${port_root} ${port_api} ${port_worker}"
      return 0
    fi
    attempt=$(( attempt + 1 ))
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
compose() {
  local worktree_name="$1"
  shift
  docker compose \
    --project-directory "${PROJECT_DIR}" \
    -f "${TEMPLATE_DIR}/docker-compose.test.yml" \
    -p "superbear-e2e-${worktree_name}" \
    --env-file "${PROJECT_DIR}/.env.e2e" \
    "$@"
}

# --- orphan 컨테이너 정리 ---
cleanup_orphans() {
  local worktree_name="$1"
  local orphans
  orphans=$(docker ps -q --filter "label=superbear.worktree=${worktree_name}" 2>/dev/null || true)
  if [[ -n "$orphans" ]]; then
    echo "Cleaning up orphaned containers from previous run..."
    echo "$orphans" | xargs -r docker rm -f 2>/dev/null || true
  fi
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

# --- 자동 정리 (trap) ---
WORKTREE_NAME=""
AUTO_CLEANUP=false

cleanup_on_exit() {
  if $AUTO_CLEANUP && [[ -n "$WORKTREE_NAME" ]]; then
    echo ""
    echo "Cleaning up e2e environment..."
    compose "$WORKTREE_NAME" down --remove-orphans --volumes 2>/dev/null || true
    rm -f "${PROJECT_DIR}/.env.e2e"
    echo "Cleanup complete"
  fi
}

trap cleanup_on_exit EXIT INT TERM

# --- 명령어 ---
cmd_up() {
  WORKTREE_NAME="$(detect_worktree)"
  echo "Worktree: ${WORKTREE_NAME}"

  # orphan 컨테이너 정리
  cleanup_orphans "$WORKTREE_NAME"

  local env_file="${PROJECT_DIR}/.env.e2e"
  if [[ -f "$env_file" ]]; then
    echo "Reusing existing .env.e2e"
    source "$env_file"
  else
    local ports
    ports="$(allocate_ports "$WORKTREE_NAME")"
    read -r port_root port_api port_worker <<< "$ports"
    write_env "$WORKTREE_NAME" "$port_root" "$port_api" "$port_worker"
    E2E_PORT_ROOT="$port_root"
    E2E_PORT_API="$port_api"
    E2E_PORT_WORKER="$port_worker"
  fi

  # 모든 서비스 기동 (postgres, redis, api, worker, root-app)
  echo "Starting all services (ports: root=${E2E_PORT_ROOT}, api=${E2E_PORT_API}, worker=${E2E_PORT_WORKER})..."
  compose "$WORKTREE_NAME" up -d --build

  # health check
  wait_for_healthy "http://localhost:${E2E_PORT_API}/api/v1/health" "API" || {
    echo "Health check failed, tearing down..."
    AUTO_CLEANUP=true
    exit 1
  }
  wait_for_healthy "http://localhost:${E2E_PORT_ROOT}" "Root App" || {
    echo "Health check failed, tearing down..."
    AUTO_CLEANUP=true
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
  WORKTREE_NAME="$(detect_worktree)"

  echo "Stopping all services (${WORKTREE_NAME})..."
  compose "$WORKTREE_NAME" down --remove-orphans --volumes 2>/dev/null || true

  rm -f "${PROJECT_DIR}/.env.e2e"
  echo "Done"
}

cmd_status() {
  WORKTREE_NAME="$(detect_worktree)"

  echo "=== E2E Services (${WORKTREE_NAME}) ==="
  compose "$WORKTREE_NAME" ps 2>/dev/null || echo "Not running"

  if [[ -f "${PROJECT_DIR}/.env.e2e" ]]; then
    echo ""
    echo "=== Ports ==="
    cat "${PROJECT_DIR}/.env.e2e"
  fi

  # orphan 감지
  local orphans
  orphans=$(docker ps -q --filter "label=superbear.e2e=true" 2>/dev/null || true)
  if [[ -n "$orphans" ]]; then
    echo ""
    echo "=== All E2E Containers ==="
    docker ps --filter "label=superbear.e2e=true" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null
  fi
}

# --- main ---
case "${1:-}" in
  up)     cmd_up ;;
  down)   cmd_down ;;
  status) cmd_status ;;
  *)
    echo "Usage: e2e-server.sh {up|down|status}"
    exit 1
    ;;
esac
