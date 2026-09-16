#!/bin/sh
set -eu

api_base="${PINYIN_API_BASE:-http://localhost:19111/api/v1}"
child_id="${PINYIN_CHILD_ID:-1}"
workspace_dir=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)

command -v curl >/dev/null 2>&1 || { echo "缺少 curl" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "缺少 jq" >&2; exit 1; }

modules=$(curl -fsS "$api_base/pinyin/modules")
module_code=$(printf '%s' "$modules" | jq -er '.data[0].code')
items=$(curl -fsS "$api_base/pinyin/modules/$module_code/items")
kp_id=$(printf '%s' "$items" | jq -er '.data[] | select(.hasGlyph or .hasSoloSpeech or .hasWordSpeech) | .kpId' | head -n 1)
curl -fsS "$api_base/pinyin/items/$kp_id" >/dev/null
has_glyph=$(printf '%s' "$items" | jq -r ".data[] | select(.kpId == $kp_id) | .hasGlyph")
if [ "$has_glyph" = "true" ]; then
  curl -fsS "$api_base/pinyin/items/$kp_id/glyph.png" >/dev/null
else
  curl -fsS "$api_base/pinyin/items/$kp_id/speech/solo.mp3" >/dev/null
fi

plan=$(curl -fsS -X POST -H 'Content-Type: application/json' -d '{"count":1}' "$api_base/children/$child_id/pinyin/plans")
plan_id=$(printf '%s' "$plan" | jq -er '.data.plan.id')
item_id=$(printf '%s' "$plan" | jq -er '.data.items[0].id')
plan_kp_id=$(printf '%s' "$plan" | jq -er '.data.items[0].kpId')
client_id="pinyin-acceptance-${plan_id}-${item_id}-$(date +%s)"

curl -fsS -X POST -H 'Content-Type: application/json' \
  -d "{\"clientId\":\"$client_id\",\"optionIndex\":0,\"costMs\":1000}" \
  "$api_base/children/$child_id/pinyin/plans/$plan_id/items/$item_id/answer" >/dev/null

sql="
SELECT 'attempts' AS table_name, COUNT(*) AS rows FROM attempts WHERE child_id=$child_id AND client_id='$client_id'
UNION ALL SELECT 'mastery_skills', COUNT(*) FROM mastery_skills WHERE child_id=$child_id AND kp_id=$plan_kp_id
UNION ALL SELECT 'mastery_states', COUNT(*) FROM mastery_states WHERE child_id=$child_id AND kp_id=$plan_kp_id
UNION ALL SELECT 'daily_stats', COUNT(*) FROM daily_stats WHERE child_id=$child_id AND stat_date=CURRENT_DATE;
"
if [ -n "${PINYIN_POSTGRES_CONTAINER:-}" ]; then
  docker exec "$PINYIN_POSTGRES_CONTAINER" psql -U conchi -d study_workbench -v ON_ERROR_STOP=1 -c "$sql"
else
  cd "$workspace_dir"
  docker compose exec -T postgres psql -U conchi -d study_workbench -v ON_ERROR_STOP=1 -c "$sql"
fi

echo "验收完成：plan=$plan_id item=$item_id kp=$plan_kp_id client=$client_id"
