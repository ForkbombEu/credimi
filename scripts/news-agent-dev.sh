#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Dev launcher for the news agent (agents/news), used by `make dev` and `make news-agent.ui`.
#   worker  run the Python worker; idles with a message when no model is configured
#   ui      run the harness session manager and web UI
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:?usage: news-agent-dev.sh worker|ui}"
cd "${ROOT_DIR}/agents/news"
set -a
# shellcheck disable=SC1091
[ ! -f .env ] || . ./.env
set +a
unset PORT
export TEMPORAL_ADDRESS="${TEMPORAL_ADDRESS:-localhost:${TEMPORAL_PORT:-7233}}"
idle() {
	echo "news agent ${mode} disabled: $1"
	while :; do sleep 3600; done
}
case "${mode}" in
worker)
	trap 'exit 0' INT TERM
	case "${CREDIMI_TEMPORAL_WORKERS_DISABLED:-}" in 1 | true | yes) idle "CREDIMI_TEMPORAL_WORKERS_DISABLED is set" ;; esac
	[ -n "${OPENAI_API_KEY:-}${NEWS_AGENT_MODEL_BASE_URL:-}${NEWS_AGENT_FAKE_MODEL:-}" ] ||
		idle "configure a model in agents/news/.env (see agents/news/.env.example)"
	until uv run python -m credimi_news_agent.worker; do
		echo "news agent worker exited; restarting in 5s"
		sleep 5
	done
	;;
ui)
	uv run temporal-agent-harness session-manager &
	SM=$!
	trap 'kill "${SM}" 2>/dev/null' EXIT INT TERM
	uv run temporal-agent-harness serve agents.toml --host 127.0.0.1 --port "${NEWS_AGENT_UI_PORT:-8000}"
	;;
*)
	echo "usage: $0 worker|ui" >&2
	exit 2
	;;
esac
