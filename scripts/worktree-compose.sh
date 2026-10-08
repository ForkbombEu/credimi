#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Shared Compose entrypoint for per-worktree infra (Temporal stack).
# Usage: scripts/worktree-compose.sh <prepare|up|stop|down|down-v|temporal-check|temporal-upgrade>
# WORKTREE_COMPOSE_MODE=docker targets the plain `docker compose` project used by
# make docker instead of the dev override.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_VERSIONS_FILE="${ROOT_DIR}/scripts/dev-compose.env"
INFRA_SERVICES=(elasticsearch postgresql temporal temporal_ui temporal_ui_embedded)
UP_SERVICES=(elasticsearch postgresql temporal temporal_ui temporal_ui_embedded temporal_schema)
PG_DATA_DIR=/var/lib/postgresql/data

# Sequential Temporal server upgrade hops: version:server-image:admin-tools-tag.
# "auto" means temporalio/auto-setup migrates the schema itself.
HOP_A="1.29.7:temporalio/auto-setup:auto"
HOP_B="1.30.7:temporalio/server:1.30.7"
HOP_C="1.31.3:temporalio/server:1.31.3"
HOP_D="1.32.0:temporalio/server:1.32.0"

usage() {
	cat <<'USAGE'
Usage: scripts/worktree-compose.sh <command>

Commands:
  prepare           Write Compose override + runtime Procfile (prints port summary)
  up                Start Temporal infra detached (implies quiet prepare + temporal-check)
  stop              Stop Temporal infra containers (implies quiet prepare)
  down              docker compose down --remove-orphans
  down-v            docker compose down -v --remove-orphans
  temporal-check    Fail if existing Temporal data needs temporal-upgrade first
  temporal-upgrade  Back up and upgrade existing Temporal data to the pinned server

Environment:
  WORKTREE_COMPOSE_MODE=docker       Use the make docker Compose project (no dev override)
  TEMPORAL_UPGRADE_FROM=1.30|1.31    Resume a failed temporal-upgrade from that hop
  TEMPORAL_UPGRADE_HOP_WAIT_SECONDS  Wait after each upgrade hop (default 600)
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
	NAMED_PG_VOLUME="${COMPOSE_PROJECT_NAME}_temporal-postgresql-data"
}

prepare() {
	local quiet="${1:-0}"
	if [[ "${quiet}" -eq 1 ]]; then
		bash "${ROOT_DIR}/scripts/worktree-dev-prepare.sh" >/dev/null
	else
		bash "${ROOT_DIR}/scripts/worktree-dev-prepare.sh"
	fi
}

docker_mode() {
	[[ "${WORKTREE_COMPOSE_MODE:-}" == "docker" ]]
}

compose() {
	if docker_mode; then
		docker compose "$@"
	else
		docker compose -f docker-compose.yaml -f "${COMPOSE_DEV_OVERRIDE_FILE}" "$@"
	fi
}

cleanup_runtime_files() {
	rm -f -- "${COMPOSE_DEV_OVERRIDE_FILE}" "${PROCFILE_RUNTIME}"
}

# Container names differ between dev and docker mode; match on Compose labels.
pg_container() {
	docker ps -aq \
		--filter "label=com.docker.compose.project=${COMPOSE_PROJECT_NAME}" \
		--filter "label=com.docker.compose.service=postgresql" | head -n1
}

pg_data_volume() {
	docker inspect -f "{{range .Mounts}}{{if eq .Destination \"${PG_DATA_DIR}\"}}{{.Name}}{{end}}{{end}}" "$1"
}

# Prints the Temporal core schema version, or nothing for a fresh database.
pg_version() {
	local has_table version
	if ! has_table="$(compose exec -T postgresql psql -U temporal -d temporal -tAc \
		"SELECT to_regclass('public.schema_version') IS NOT NULL")"; then
		echo "error: cannot query the Temporal Postgres database" >&2
		exit 1
	fi
	if [[ "$(tr -d '[:space:]' <<<"${has_table}")" != "t" ]]; then
		return 0
	fi
	if ! version="$(compose exec -T postgresql psql -U temporal -d temporal -tAc \
		"SELECT curr_version FROM schema_version WHERE db_name='temporal'")"; then
		echo "error: cannot read the Temporal schema version" >&2
		exit 1
	fi
	tr -d '[:space:]' <<<"${version}"
}

upgrade_command() {
	if docker_mode; then
		echo "WORKTREE_COMPOSE_MODE=docker ./scripts/worktree-compose.sh temporal-upgrade"
	else
		echo "./scripts/worktree-compose.sh prepare && ./scripts/worktree-compose.sh temporal-upgrade"
	fi
}

# temporal-check and temporal-upgrade never start the credimi service, but
# Compose still interpolates its required variables; give unset ones a placeholder.
credimi_placeholder_env() {
	export PUBLIC_TURNSTILE_SITE_KEY="${PUBLIC_TURNSTILE_SITE_KEY:-unused-by-temporal-upgrade}"
	export TURNSTILE_SECRET_KEY="${TURNSTILE_SECRET_KEY:-unused-by-temporal-upgrade}"
	export CREDIMI_RUNNER_CREDENTIAL_SECRET="${CREDIMI_RUNNER_CREDENTIAL_SECRET:-unused-by-temporal-upgrade}"
	export CREDIMI_TEMPORAL_SECRETS_ENCRYPTION_KEY="${CREDIMI_TEMPORAL_SECRETS_ENCRYPTION_KEY:-unused-by-temporal-upgrade}"
}

temporal_check() {
	local container v
	container="$(pg_container)"
	if [[ -n "${container}" && "$(pg_data_volume "${container}")" != "${NAMED_PG_VOLUME}" ]]; then
		echo "Temporal data is on the old anonymous volume. Run: $(upgrade_command)" >&2
		exit 1
	fi
	if ! docker volume inspect "${NAMED_PG_VOLUME}" >/dev/null 2>&1; then
		return 0
	fi
	compose up -d --wait postgresql
	v="$(pg_version)"
	if [[ -n "${v}" && "$(printf '%s\n1.19\n' "${v}" | sort -V | head -n1)" != "1.19" ]]; then
		echo "Temporal schema ${v} predates server 1.31. Run the temporal-upgrade subcommand of scripts/worktree-compose.sh" >&2
		exit 1
	fi
}

copy_volume() {
	docker run --rm -v "$1":/from:ro -v "$2":/to alpine sh -c 'cp -a /from/. /to/'
}

UPGRADE_BACKUPS=()

print_backups() {
	local backup
	echo "Backup volumes:"
	for backup in "${UPGRADE_BACKUPS[@]}"; do
		echo "  ${backup}"
	done
	cat <<EOF
To restore: stop the stack, copy each backup back over its volume, then docker compose up:
  docker run --rm -v <backup>:/from:ro -v <volume>:/to alpine sh -c 'find /to -mindepth 1 -delete && cp -a /from/. /to/'
EOF
}

on_upgrade_exit() {
	local status="$1"
	if [[ "${status}" -ne 0 ]]; then
		echo "Temporal upgrade failed; existing data is preserved in the backups." >&2
		print_backups >&2
	fi
}

backup_volume() {
	local volume="$1" backup="$2"
	if ! docker volume inspect "${volume}" >/dev/null 2>&1; then
		echo "warning: volume ${volume} not found; skipping its backup" >&2
		return 0
	fi
	docker volume create "${backup}" >/dev/null
	copy_volume "${volume}" "${backup}"
	UPGRADE_BACKUPS+=("${backup}")
}

migrate_pg_volume() {
	local container="$1" from pg_major
	from="$(pg_data_volume "${container}")"
	if [[ -z "${from}" ]]; then
		echo "error: no Postgres data volume found on container ${container}" >&2
		exit 1
	fi
	echo "Copying Temporal Postgres data from volume ${from} to ${NAMED_PG_VOLUME}..."
	docker volume create \
		--label "com.docker.compose.project=${COMPOSE_PROJECT_NAME}" \
		--label com.docker.compose.volume=temporal-postgresql-data \
		"${NAMED_PG_VOLUME}" >/dev/null
	copy_volume "${from}" "${NAMED_PG_VOLUME}"
	pg_major="$(docker run --rm -v "${NAMED_PG_VOLUME}":/d alpine cat /d/PG_VERSION)"
	if [[ "${pg_major}" != "${POSTGRESQL_VERSION}" ]]; then
		echo "error: copied PG_VERSION is '${pg_major}', expected ${POSTGRESQL_VERSION}; old volume ${from} is untouched" >&2
		exit 1
	fi
	compose rm -f postgresql
}

temporal_serving() {
	local out
	out="$(TEMPORAL_ADMIN_TOOLS_VERSION="${HOP_D##*:}" compose run --rm -T --no-deps --entrypoint temporal \
		temporal_schema operator cluster health --address temporal:7233 2>/dev/null || true)"
	[[ "${out}" == *SERVING* ]]
}

run_hop() {
	local version image tools i
	IFS=: read -r version image tools <<<"$1"
	echo "==> Temporal ${version} (${image})"
	if [[ "${tools}" != "auto" ]]; then
		TEMPORAL_ADMIN_TOOLS_VERSION="${tools}" compose run --rm temporal_schema
	fi
	TEMPORAL_SERVER_IMAGE="${image}" TEMPORAL_VERSION="${version}" \
		compose up -d --no-deps --force-recreate temporal
	for ((i = 0; i < 60; i++)); do
		if temporal_serving; then
			break
		fi
		sleep 5
	done
	if [[ "${i}" -eq 60 ]]; then
		echo "error: Temporal ${version} did not report SERVING within 5 minutes" >&2
		exit 1
	fi
	echo "Temporal ${version} is SERVING; waiting ${TEMPORAL_UPGRADE_HOP_WAIT_SECONDS:-600}s for history shards to update..."
	sleep "${TEMPORAL_UPGRADE_HOP_WAIT_SECONDS:-600}"
}

temporal_upgrade() {
	local container ts v hop
	local -a hops
	compose stop
	# The auto-setup era temporal_setup job no longer exists in the compose file.
	docker ps -aq \
		--filter "label=com.docker.compose.project=${COMPOSE_PROJECT_NAME}" \
		--filter label=com.docker.compose.service=temporal_setup | xargs -r docker rm >/dev/null

	container="$(pg_container)"
	if [[ -n "${container}" && "$(pg_data_volume "${container}")" != "${NAMED_PG_VOLUME}" ]]; then
		migrate_pg_volume "${container}"
	elif [[ -z "${container}" ]] && ! docker volume inspect "${NAMED_PG_VOLUME}" >/dev/null 2>&1; then
		echo "No existing Temporal data; nothing to upgrade."
		exit 0
	fi

	ts="$(date +%Y%m%d%H%M%S)"
	backup_volume "${NAMED_PG_VOLUME}" "${NAMED_PG_VOLUME}-backup-${ts}"
	backup_volume "${COMPOSE_PROJECT_NAME}_esdata" "${COMPOSE_PROJECT_NAME}_esdata-backup-${ts}"
	print_backups
	trap 'on_upgrade_exit $?' EXIT

	compose up -d --wait postgresql elasticsearch
	v="$(pg_version)"
	case "${v}|${TEMPORAL_UPGRADE_FROM:-}" in
	"|"*)
		echo "No Temporal schema found; nothing to upgrade."
		exit 0
		;;
	"1.18|") hops=("${HOP_A}" "${HOP_B}" "${HOP_C}" "${HOP_D}") ;;
	"1.18|1.30") hops=("${HOP_B}" "${HOP_C}" "${HOP_D}") ;;
	"1.19|") hops=("${HOP_D}") ;;
	"1.19|1.31") hops=("${HOP_C}" "${HOP_D}") ;;
	*)
		echo "error: unsupported Temporal schema version '${v}' (TEMPORAL_UPGRADE_FROM=${TEMPORAL_UPGRADE_FROM:-unset})" >&2
		exit 1
		;;
	esac

	for hop in "${hops[@]}"; do
		run_hop "${hop}"
	done
	echo "Temporal upgraded to 1.32.0. Start the stack with make dev (or make docker)."
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
		temporal_check
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
	temporal-check)
		load_env
		prepare 1
		credimi_placeholder_env
		temporal_check
		;;
	temporal-upgrade)
		load_env
		prepare 1
		credimi_placeholder_env
		temporal_upgrade
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
