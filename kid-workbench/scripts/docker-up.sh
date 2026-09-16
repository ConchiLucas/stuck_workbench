#!/usr/bin/env bash
# Start every kid-workbench frontend and backend with Docker Compose.
#
#   ./scripts/docker-up.sh           # build images, then start
#   ./scripts/docker-up.sh --no-build
#   ./scripts/docker-up.sh --down    # stop the stack
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BUILD=1
ACTION=up

usage() {
  cat <<'EOF'
Usage: ./scripts/docker-up.sh [--no-build] [--down]

  (default)  docker compose up --build -d
  --no-build skip image rebuild
  --down     docker compose down --remove-orphans
EOF
}

for arg in "$@"; do
  case "$arg" in
    --no-build) BUILD=0 ;;
    --down) ACTION=down ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown flag: $arg" >&2; usage >&2; exit 1 ;;
  esac
done

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "need $1 on PATH" >&2
    exit 1
  }
}

need_cmd docker
docker info >/dev/null 2>&1 || {
  echo "Docker daemon is not running. Start Docker Desktop, then retry." >&2
  exit 1
}

ensure_shared_network() {
  docker network inspect vibedeploy-shared >/dev/null 2>&1 && return
  echo "creating docker network vibedeploy-shared"
  docker network create vibedeploy-shared
}

start_shared_config_center() {
  local dir="${SHARED_CONFIG_CENTER_DIR:-$ROOT/../../go_workforce/shared-config-center}"
  if [[ ! -f "$dir/docker-compose.yml" ]]; then
    echo "skip shared-config-center (not found at $dir)"
    echo "  content-admin config menu needs it; set SHARED_CONFIG_CENTER_DIR if it lives elsewhere"
    return
  fi
  echo "starting shared-config-center"
  docker compose -f "$dir/docker-compose.yml" up -d
}

print_urls() {
  cat <<'EOF'

kid-workbench is up (Docker):

  进度后台              http://localhost:19081
  素材后台              http://localhost:19091
  题目后台              http://localhost:19201
  孩子知识库            http://localhost:19211

  识字 API / 孩子端     http://localhost:19151  http://localhost:19152
  拼音 API / 孩子端     http://127.0.0.1:19111  http://localhost:19112
  科普 API / 孩子端     http://localhost:19121  http://localhost:19122
  英语 API / 孩子端     http://localhost:19131  http://localhost:19132
  算数 API / 孩子端     http://localhost:19141  http://localhost:19142
  古诗 API / 孩子端     http://localhost:19161  http://localhost:19162
  英语短句 API / 孩子端 http://localhost:19171  http://localhost:19172
  成语 API / 孩子端     http://localhost:19181  http://localhost:19182
  逻辑 API / 孩子端     http://localhost:19191  http://localhost:19192

  PostgreSQL            localhost:15432

Stop:  ./scripts/docker-up.sh --down
Logs:  docker compose logs -f
EOF
}

check_http() {
  local url="$1"
  curl -fsS -o /dev/null --max-time 3 "$url" && echo "  ok  $url" || echo "  ..  $url (not ready yet)"
}

if [[ "$ACTION" == down ]]; then
  echo "stopping kid-workbench"
  docker compose down --remove-orphans
  echo "stopped"
  exit 0
fi

ensure_shared_network
start_shared_config_center

echo "starting kid-workbench compose stack"
if [[ "$BUILD" -eq 1 ]]; then
  docker compose up --build -d
else
  docker compose up -d
fi

echo
docker compose ps
print_urls

echo "health:"
check_http http://localhost:19081/healthz
check_http http://localhost:19091/healthz
check_http http://localhost:19201/healthz
check_http http://localhost:19211/healthz
check_http http://localhost:19151/healthz
check_http http://localhost:19152/
check_http http://127.0.0.1:19111/healthz
check_http http://localhost:19121/healthz
check_http http://localhost:19131/healthz
check_http http://localhost:19141/healthz
check_http http://localhost:19161/healthz
check_http http://localhost:19171/healthz
check_http http://localhost:19181/healthz
check_http http://localhost:19191/healthz
