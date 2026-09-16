#!/bin/sh
set -eu

base_url="${MATH_BASE_URL:-http://localhost:19141/api/v1}"
child_id="${MATH_CHILD_ID:-1}"
client_id="math-acceptance-$(date +%s)-$$"

curl -fsS "${base_url}/math/modules"
plan_json="$(curl -fsS -X POST -H 'Content-Type: application/json' \
  -d '{"kind":"module","moduleCode":"add10","stageCode":"within5"}' \
  "${base_url}/children/${child_id}/math/plans")"

plan_id="$(printf '%s' "$plan_json" | sed -n 's/.*"plan"[^{]*{[^}]*"id":\([0-9][0-9]*\).*/\1/p')"
item_id="$(printf '%s' "$plan_json" | sed -n 's/.*"items"[^{]*{[^}]*"itemId":\([0-9][0-9]*\).*/\1/p')"
test -n "$plan_id"
test -n "$item_id"

curl -fsS -X POST "${base_url}/children/${child_id}/math/plans/${plan_id}/start"
curl -fsS -X POST -H 'Content-Type: application/json' \
  -d "{\"clientId\":\"${client_id}\",\"optionIndex\":0,\"costMs\":1000}" \
  "${base_url}/children/${child_id}/math/plans/${plan_id}/items/${item_id}/answer"
curl -fsS "${base_url}/children/${child_id}/math/plans/${plan_id}"
