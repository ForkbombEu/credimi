#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULTS_FILE="${ROOT_DIR}/scripts/dev-ports.env"
WORKTREE_ENV_FILE="${ROOT_DIR}/.env.worktree"

sanitize_compose_project_name() {
	basename "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-$(sanitize_compose_project_name "${ROOT_DIR}")}"

usage() {
	cat <<'USAGE'
Usage: scripts/worktree-env.sh <command>

Commands:
  print         Print resolved port env (defaults + .env.worktree overrides)
  export        Same as print, each line prefixed with `export ` (for eval)
  project-name  Print sanitized COMPOSE_PROJECT_NAME for this checkout
  write         Write .env.worktree if missing (unique ports via Worktrunk hash_port;
                never overwrites). Requires `wt` unless --classic.
  write --classic
                Write classic defaults from scripts/dev-ports.env if missing
  sync-urls     Rewrite PUBLIC_POCKETBASE_URL / VITE_API / PB_TYPEGEN_URL in webapp/.env
USAGE
}

load_defaults() {
	set -a
	# shellcheck disable=SC1091
	source "${DEFAULTS_FILE}"
	if [[ -f "${WORKTREE_ENV_FILE}" ]]; then
		# shellcheck disable=SC1091
		source "${WORKTREE_ENV_FILE}"
	fi
	set +a
}

resolve_runtime_paths() {
	COMPOSE_DEV_OVERRIDE_FILE="${COMPOSE_DEV_OVERRIDE_FILE:-/tmp/${COMPOSE_PROJECT_NAME}-docker-compose.dev.yaml}"
	PROCFILE_RUNTIME="${PROCFILE_RUNTIME:-/tmp/${COMPOSE_PROJECT_NAME}-Procfile.dev}"
}

port_free() {
	local port="$1"
	if command -v lsof >/dev/null 2>&1; then
		! lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
	else
		! (echo >/dev/tcp/127.0.0.1/"${port}") 2>/dev/null
	fi
}

# Seed from Worktrunk's hash_port (10000-19999), then walk on collision.
seed_port() {
	local label="$1"
	if ! command -v wt >/dev/null 2>&1; then
		echo "error: Worktrunk (\`wt\`) is required to allocate unique ports" >&2
		exit 1
	fi
	wt step eval "{{ (\"${label}-\" ~ branch) | hash_port }}"
}

pick_port() {
	local label="$1"
	local candidate
	candidate="$(seed_port "${label}")"
	local i=0
	while [[ "${i}" -lt 200 ]]; do
		if port_free "${candidate}"; then
			echo "${candidate}"
			return 0
		fi
		candidate=$((10000 + (candidate - 10000 + 1) % 10000))
		i=$((i + 1))
	done
	echo "error: could not find a free port for label ${label}" >&2
	return 1
}

branch_name() {
	git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "detached"
}

print_env() {
	load_defaults
	resolve_runtime_paths
	cat <<EOF
API_PORT=${API_PORT}
UI_PORT=${UI_PORT}
TEMPORAL_PORT=${TEMPORAL_PORT}
TEMPORAL_UI_PORT=${TEMPORAL_UI_PORT}
ADDRESS_UI=http://localhost:${UI_PORT}
TEMPORAL_ADDRESS=localhost:${TEMPORAL_PORT}
CREDIMI_INTERNAL_APP_URL=http://localhost:${API_PORT}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME}
COMPOSE_DEV_OVERRIDE_FILE=${COMPOSE_DEV_OVERRIDE_FILE}
PROCFILE_RUNTIME=${PROCFILE_RUNTIME}
EOF
}

cmd_print() {
	print_env
}

cmd_export() {
	print_env | sed 's/^/export /'
}

cmd_project_name() {
	echo "${COMPOSE_PROJECT_NAME}"
}

write_ports_file() {
	local header="$1"
	cat >"${WORKTREE_ENV_FILE}" <<EOF
${header}
API_PORT=${API_PORT}
UI_PORT=${UI_PORT}
TEMPORAL_PORT=${TEMPORAL_PORT}
TEMPORAL_UI_PORT=${TEMPORAL_UI_PORT}
EOF
}

cmd_write() {
	local classic=0
	if [[ "${1:-}" == "--classic" ]]; then
		classic=1
	fi
	if [[ -f "${WORKTREE_ENV_FILE}" ]]; then
		echo "keeping existing ${WORKTREE_ENV_FILE}"
		return 0
	fi
	# shellcheck disable=SC1091
	source "${DEFAULTS_FILE}"
	if [[ "${classic}" -eq 1 ]]; then
		write_ports_file "# Generated classic ports from scripts/dev-ports.env"
	else
		local branch
		branch="$(branch_name)"
		API_PORT="$(pick_port "api")"
		UI_PORT="$(pick_port "ui")"
		TEMPORAL_PORT="$(pick_port "temporal")"
		TEMPORAL_UI_PORT="$(pick_port "temporal-ui")"
		write_ports_file "# Generated unique ports for worktree ${COMPOSE_PROJECT_NAME} (branch ${branch})
# Seeds from Worktrunk hash_port; collision walk applied. Edit freely; not overwritten."
	fi
	echo "wrote ${WORKTREE_ENV_FILE}"
}

cmd_sync_urls() {
	load_defaults
	local webenv="${ROOT_DIR}/webapp/.env"
	if [[ ! -f "${webenv}" ]]; then
		echo "skip sync-urls: ${webenv} missing"
		return 0
	fi
	local api_url="http://127.0.0.1:${API_PORT}"
	local tmp
	tmp="$(mktemp)"
	awk -v api="${api_url}" '
		BEGIN { done_pb=0; done_vite=0; done_typegen=0 }
		/^PUBLIC_POCKETBASE_URL=/ { print "PUBLIC_POCKETBASE_URL=" api "/"; done_pb=1; next }
		/^VITE_API=/ { print "VITE_API=" api; done_vite=1; next }
		/^PB_TYPEGEN_URL=/ { print "PB_TYPEGEN_URL=" api; done_typegen=1; next }
		{ print }
		END {
			if (!done_pb) print "PUBLIC_POCKETBASE_URL=" api "/"
			if (!done_vite) print "VITE_API=" api
			if (!done_typegen) print "PB_TYPEGEN_URL=" api
		}
	' "${webenv}" >"${tmp}"
	mv "${tmp}" "${webenv}"
	echo "synced PocketBase URLs in webapp/.env -> ${api_url}"
}

main() {
	local cmd="${1:-}"
	shift || true
	case "${cmd}" in
	print) cmd_print ;;
	export) cmd_export ;;
	project-name) cmd_project_name ;;
	write) cmd_write "$@" ;;
	sync-urls) cmd_sync_urls ;;
	""|-h|--help) usage ;;
	*)
		echo "unknown command: ${cmd}" >&2
		usage >&2
		exit 1
		;;
	esac
}

main "$@"
