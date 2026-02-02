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

# Wait for any process to exit
wait -n "$preview_pid" "$mock_pid"
exit_status=$?

# Check which process(es) exited
preview_running=false
mock_running=false

if kill -0 "$preview_pid" 2>/dev/null; then
	preview_running=true
fi

if kill -0 "$mock_pid" 2>/dev/null; then
	mock_running=true
fi

# Report which process failed
if ! $preview_running && ! $mock_running; then
	echo "ERROR: Both servers have stopped unexpectedly" >&2
	exit 1
elif ! $preview_running; then
	echo "ERROR: Preview server (PID $preview_pid) has stopped unexpectedly" >&2
	exit 1
elif ! $mock_running; then
	echo "ERROR: Mock PocketBase server (PID $mock_pid) has stopped unexpectedly" >&2
	exit 1
fi

# If we reach here, wait returned but both processes are still running (unexpected)
echo "WARNING: wait -n returned but both processes are still running (exit status: $exit_status)" >&2
exit "$exit_status"
