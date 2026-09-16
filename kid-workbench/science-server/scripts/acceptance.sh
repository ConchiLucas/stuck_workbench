#!/bin/sh
set -eu

base_url="${SCIENCE_BASE_URL:-http://localhost:19121/api/v1}"
child_id="${SCIENCE_CHILD_ID:-1}"
client_id="science-acceptance-$(date +%s)-$$"

curl -fsS "${base_url}/science/modules"
plan_json="$(curl -fsS -X POST -H 'Content-Type: application/json' \
  -d '{"mode":"daily"}' "${base_url}/children/${child_id}/science/plans")"

plan_id="$(printf '%s' "$plan_json" | sed -n 's/.*"plan"[^{]*{[^}]*"id":\([0-9][0-9]*\).*/\1/p')"
item_id="$(printf '%s' "$plan_json" | sed -n 's/.*"items"[^{]*{[^}]*"id":\([0-9][0-9]*\).*/\1/p')"
test -n "$plan_id"
test -n "$item_id"

curl -fsS -X POST "${base_url}/children/${child_id}/science/plans/${plan_id}/start"
curl -fsS -X POST -H 'Content-Type: application/json' \
  -d "{\"clientId\":\"${client_id}\",\"optionIndex\":0,\"costMs\":1000}" \
  "${base_url}/children/${child_id}/science/plans/${plan_id}/items/${item_id}/answer"
curl -fsS "${base_url}/children/${child_id}/science/plans/${plan_id}"
