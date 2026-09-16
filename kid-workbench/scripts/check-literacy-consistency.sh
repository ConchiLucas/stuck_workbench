#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export GOCACHE="${GOCACHE:-/tmp/kid-workbench-go-cache}"
node scripts/generate-literacy-contracts.mjs --check
node --test scripts/generate-literacy-contracts.test.mjs scripts/literacy-host-consistency.test.mjs
npm --prefix packages/literacy-player test
for frontend in literacy-app content-admin/frontend task-admin/frontend; do
  (cd "$frontend" && npm test && npm run build)
done
for backend in shared-go content-admin/backend task-admin/backend literacy-server parent-dashboard/backend diagnosis-admin/backend; do
  (cd "$backend" && go test ./...)
done
