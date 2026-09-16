#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Shared Compose entrypoint for per-worktree infra (Temporal stack).
# Usage: scripts/worktree-compose.sh <prepare|up|stop|down|down-v>
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_VERSIONS_FILE="${ROOT_DIR}/scripts/dev-compose.env"
INFRA_SERVICES=(elasticsearch postgresql temporal temporal_ui)
UP_SERVICES=(elasticsearch postgresql temporal temporal_ui temporal_setup)

usage() {
	cat <<'USAGE'
Usage: scripts/worktree-compose.sh <command>

Commands:
  prepare  Write Compose override + runtime Procfile (prints port summary)
  up       Start Temporal infra detached (implies quiet prepare)
  stop     Stop Temporal infra containers (implies quiet prepare)
  down     docker compose down --remove-orphans
  down-v   docker compose down -v --remove-orphans
USAGE
}

load_env() {
	cd "${ROOT_DIR}"
	# shellcheck disable=SC1091
	eval "$(bash "${ROOT_DIR}/scripts/worktree-env.sh" export)"
	export CREDIMI_ELASTIC_PASSWORD="${CREDIMI_ELASTIC_PASSWORD:-devpassword}"
	set -a
	# shellcheck disable=SC1091
	source "${COMPOSE_VERSIONS_FILE}"
	set +a
}

prepare() {
	local quiet="${1:-0}"
	if [[ "${quiet}" -eq 1 ]]; then
		bash "${ROOT_DIR}/scripts/worktree-dev-prepare.sh" >/dev/null
	else
		bash "${ROOT_DIR}/scripts/worktree-dev-prepare.sh"
	fi
}

compose() {
	docker compose -f docker-compose.yaml -f "${COMPOSE_DEV_OVERRIDE_FILE}" "$@"
}

cleanup_runtime_files() {
	rm -f -- "${COMPOSE_DEV_OVERRIDE_FILE}" "${PROCFILE_RUNTIME}"
}

main() {
	local cmd="${1:-}"
	case "${cmd}" in
	prepare)
		load_env
		prepare 0
		;;
	up)
		load_env
		prepare 1
		compose up --build -d "${UP_SERVICES[@]}"
		;;
	stop)
		load_env
		prepare 1
		compose stop "${INFRA_SERVICES[@]}"
		;;
	down)
		load_env
		prepare 1
		compose down --remove-orphans
		;;
	down-v)
		load_env
		prepare 1
		compose down -v --remove-orphans
		cleanup_runtime_files
		;;
	"" | -h | --help)
		usage
		;;
	*)
		echo "unknown command: ${cmd}" >&2
		usage >&2
		exit 1
		;;
	esac
}

main "$@"
