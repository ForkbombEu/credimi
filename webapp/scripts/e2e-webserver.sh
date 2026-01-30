#!/usr/bin/env bash
set -euo pipefail

: "${PUBLIC_POCKETBASE_URL:=http://127.0.0.1:8090}"
export PUBLIC_POCKETBASE_URL

bun run build
bun run preview --port 5100 &
preview_pid=$!

bun run ./scripts/mock-pocketbase.ts &
mock_pid=$!

cleanup() {
	kill "$preview_pid" "$mock_pid" 2>/dev/null || true
}
trap cleanup EXIT

wait -n "$preview_pid" "$mock_pid"
