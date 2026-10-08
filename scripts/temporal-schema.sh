#!/bin/sh
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Creates or upgrades the Temporal Postgres and Elasticsearch schemas.
# Runs inside temporalio/admin-tools (Alpine, no bash) as the temporal_schema
# Compose job. Every command is idempotent: safe on fresh and existing data.
# Credentials come from SQL_PASSWORD, ES_USER and ES_PWD.
set -eu

sql() {
	temporal-sql-tool --plugin postgres12 --ep "${SQL_HOST}" -p "${SQL_PORT}" -u "${SQL_USER}" --db temporal "$@"
}

es() {
	temporal-elasticsearch-tool --ep "${ES_SERVER}" "$@"
}

until nc -z "${SQL_HOST}" "${SQL_PORT}"; do sleep 1; done
until es ping >/dev/null 2>&1; do sleep 2; done

sql create
sql setup-schema -v 0.0
sql update-schema -d /etc/temporal/schema/postgresql/v12/temporal/versioned

es setup-schema
es create-index --index "${ES_VISIBILITY_INDEX}"
es update-schema --index "${ES_VISIBILITY_INDEX}"
