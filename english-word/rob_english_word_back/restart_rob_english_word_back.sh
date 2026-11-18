#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ROOT_DIR/restart_service_common.sh"

SERVICE_NAME="rob_english_word_back"
SERVICE_DIR="$ROOT_DIR/rob_english_word_back"
API_PORT=8019
WS_PORT=9091

need_cmd mvn

restart_service "$SERVICE_NAME" "$API_PORT" "$WS_PORT"
start_process "$SERVICE_NAME" "$SERVICE_DIR" \
  env \
    SPRING_DATASOURCE_URL="${SPRING_DATASOURCE_URL:-jdbc:postgresql://100.120.144.76:5432/study_workbench}" \
    SPRING_DATASOURCE_USERNAME="${SPRING_DATASOURCE_USERNAME:?load the workspace runtime environment}}" \
    SPRING_DATASOURCE_PASSWORD="${SPRING_DATASOURCE_PASSWORD:?load the workspace runtime environment}}" \
    SPRING_REDIS_HOST="${SPRING_REDIS_HOST:-127.0.0.1}" \
    SPRING_DATA_REDIS_HOST="${SPRING_DATA_REDIS_HOST:-127.0.0.1}" \
    SPRING_DATA_REDIS_PASSWORD="${SPRING_DATA_REDIS_PASSWORD:-${REDIS_PASSWORD:-conchi123456}}" \
    WORD_AGENT_BASE_URL="${WORD_AGENT_BASE_URL:-http://127.0.0.1:8010}" \
    mvn spring-boot:run

wait_for_port "$SERVICE_NAME" "$API_PORT" 120
wait_for_port "${SERVICE_NAME}_websocket" "$WS_PORT" 120
log "API: http://127.0.0.1:$API_PORT"
log "WebSocket: ws://127.0.0.1:$WS_PORT"
