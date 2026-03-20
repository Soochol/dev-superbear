#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="$ROOT_DIR/.dev-logs"
mkdir -p "$LOG_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}[dev]${NC} $1"; }
ok()  { echo -e "${GREEN}[dev]${NC} $1"; }
warn(){ echo -e "${YELLOW}[dev]${NC} $1"; }
err() { echo -e "${RED}[dev]${NC} $1"; }

# ─── Kill existing processes ───
kill_all() {
    log "Stopping existing services..."

    # Kill frontend (port 3000)
    if lsof -ti:3000 &>/dev/null; then
        kill $(lsof -ti:3000) 2>/dev/null || true
        ok "  Frontend killed (port 3000)"
    fi

    # Kill backend (port 8080)
    if lsof -ti:8080 &>/dev/null; then
        kill $(lsof -ti:8080) 2>/dev/null || true
        ok "  Backend killed (port 8080)"
    fi

    # Stop PostgreSQL container
    if docker ps -q --filter "name=nexus-db" 2>/dev/null | grep -q .; then
        docker stop nexus-db &>/dev/null || true
        ok "  PostgreSQL container stopped"
    fi
}

# ─── Start PostgreSQL ───
start_db() {
    log "Starting PostgreSQL..."

    if docker ps -q --filter "name=nexus-db" 2>/dev/null | grep -q .; then
        ok "  PostgreSQL already running"
        return
    fi

    # Remove stopped container if exists
    docker rm nexus-db &>/dev/null || true

    # Check if an existing PostgreSQL is already on port 5432
    if docker ps --filter "publish=5432" --format "{{.Names}}" 2>/dev/null | grep -q .; then
        EXISTING_DB=$(docker ps --filter "publish=5432" --format "{{.Names}}")
        warn "  Using existing PostgreSQL container: $EXISTING_DB"
        # Ensure nexus user/db exist
        ADMIN_USER=$(docker inspect "$EXISTING_DB" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep POSTGRES_USER | cut -d= -f2)
        ADMIN_DB=$(docker inspect "$EXISTING_DB" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep POSTGRES_DB | cut -d= -f2)
        docker exec "$EXISTING_DB" psql -U "$ADMIN_USER" -d "$ADMIN_DB" -c "CREATE USER nexus WITH PASSWORD 'nexus';" 2>/dev/null || true
        docker exec "$EXISTING_DB" psql -U "$ADMIN_USER" -d "$ADMIN_DB" -c "CREATE DATABASE nexus OWNER nexus;" 2>/dev/null || true
        ok "  PostgreSQL ready (port 5432, container: $EXISTING_DB)"
        return
    fi

    docker run -d \
        --name nexus-db \
        -e POSTGRES_USER=nexus \
        -e POSTGRES_PASSWORD=nexus \
        -e POSTGRES_DB=nexus \
        -p 5432:5432 \
        postgres:16-alpine \
        > /dev/null

    # Wait for PostgreSQL to be ready
    log "  Waiting for PostgreSQL..."
    for i in $(seq 1 30); do
        if docker exec nexus-db pg_isready -U nexus &>/dev/null; then
            ok "  PostgreSQL ready (port 5432)"
            return
        fi
        sleep 1
    done
    err "  PostgreSQL failed to start"
    exit 1
}

# ─── Run migrations ───
run_migrations() {
    log "Running migrations..."
    docker exec -i nexus-db psql -U nexus -d nexus < "$ROOT_DIR/backend/db/migrations/001_initial.sql" 2>/dev/null && \
        ok "  Migrations applied" || \
        warn "  Migrations skipped (already applied or error)"
}

# ─── Start backend ───
start_backend() {
    log "Starting Go backend..."
    cd "$ROOT_DIR/backend"

    export DATABASE_URL="postgresql://nexus:nexus@localhost:5432/nexus?sslmode=disable"
    export JWT_SECRET="dev-secret-change-in-production"
    export APP_ENV="development"
    export ALLOWED_ORIGIN="http://localhost:3000"
    export PORT="8080"

    go run ./cmd/api/ > "$LOG_DIR/backend.log" 2>&1 &
    echo $! > "$LOG_DIR/backend.pid"

    sleep 2
    if curl -s http://localhost:8080/api/v1/health | grep -q "ok"; then
        ok "  Backend running (port 8080) — PID $(cat "$LOG_DIR/backend.pid")"
    else
        warn "  Backend started but health check pending"
    fi
}

# ─── Start frontend ───
start_frontend() {
    log "Starting Next.js frontend..."
    cd "$ROOT_DIR"

    NEXT_PUBLIC_API_URL="http://localhost:8080" \
        npm run dev -- --port 3000 > "$LOG_DIR/frontend.log" 2>&1 &
    echo $! > "$LOG_DIR/frontend.pid"

    sleep 3
    if curl -s http://localhost:3000 | grep -q "NEXUS"; then
        ok "  Frontend running (port 3000) — PID $(cat "$LOG_DIR/frontend.pid")"
    else
        warn "  Frontend started but still initializing"
    fi
}

# ─── Status ───
status() {
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo -e "${CYAN}  NEXUS Development Environment${NC}"
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo ""
    echo -e "  ${GREEN}▸${NC} PostgreSQL  http://localhost:5432"
    echo -e "  ${GREEN}▸${NC} Backend     http://localhost:8080/api/v1/health"
    echo -e "  ${GREEN}▸${NC} Frontend    http://localhost:3000"
    echo ""
    echo -e "  Logs: ${YELLOW}$LOG_DIR/${NC}"
    echo -e "  Stop: ${YELLOW}$0 stop${NC}"
    echo ""
}

# ─── Main ───
case "${1:-start}" in
    start)
        kill_all
        start_db
        run_migrations
        start_backend
        start_frontend
        status
        ;;
    stop)
        kill_all
        ok "All services stopped."
        ;;
    restart)
        kill_all
        start_db
        run_migrations
        start_backend
        start_frontend
        status
        ;;
    db)
        start_db
        run_migrations
        ;;
    logs)
        echo "=== Backend ===" && tail -20 "$LOG_DIR/backend.log" 2>/dev/null
        echo "=== Frontend ===" && tail -20 "$LOG_DIR/frontend.log" 2>/dev/null
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|db|logs}"
        exit 1
        ;;
esac
