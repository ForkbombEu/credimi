#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Writes the per-worktree Compose host-port override and runtime Procfile.
# Expects port env from scripts/worktree-env.sh print (or exports).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
eval "$(bash "${ROOT_DIR}/scripts/worktree-env.sh" export)"

printf '%s\n' \
	'services:' \
	'  elasticsearch:' \
	"    container_name: ${COMPOSE_PROJECT_NAME}-temporal-elasticsearch" \
	'  postgresql:' \
	"    container_name: ${COMPOSE_PROJECT_NAME}-temporal-postgresql" \
	'  temporal:' \
	'  temporal_ui:' \
	"    container_name: ${COMPOSE_PROJECT_NAME}-temporal-ui" \
	>"${COMPOSE_DEV_OVERRIDE_FILE}"

printf '%s\n' \
	"# Generated runtime Procfile for ${COMPOSE_PROJECT_NAME}; do not edit." \
	"API: ./scripts/wait-for-it.sh -s -t 0 localhost:${TEMPORAL_PORT} && go tool gow run -tags=credimi_extra main.go serve --http=0.0.0.0:${API_PORT}" \
	"UI: ./scripts/wait-for-it.sh -s -t 0 localhost:${API_PORT} && cd webapp && bun i && PORT=${UI_PORT} bun dev" \
	>"${PROCFILE_RUNTIME}"

cat <<EOF
worktree ports (${COMPOSE_PROJECT_NAME}):
  app         http://localhost:${API_PORT}
  ui          http://localhost:${UI_PORT}  (proxied via app)
  temporal    localhost:${TEMPORAL_PORT}
  temporal-ui http://localhost:${TEMPORAL_UI_PORT}
  override    ${COMPOSE_DEV_OVERRIDE_FILE}
  procfile    ${PROCFILE_RUNTIME}
EOF
