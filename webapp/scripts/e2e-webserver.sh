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

# Wait for any process to exit and check which one failed
wait -n "$preview_pid" "$mock_pid"
exit_status=$?

# Check which process exited
if ! kill -0 "$preview_pid" 2>/dev/null; then
	echo "ERROR: Preview server (PID $preview_pid) has exited with status $exit_status" >&2
	exit 1
elif ! kill -0 "$mock_pid" 2>/dev/null; then
	echo "ERROR: Mock PocketBase server (PID $mock_pid) has exited with status $exit_status" >&2
	exit 1
fi

# If we reach here, a process exited successfully (shouldn't normally happen)
exit "$exit_status"
