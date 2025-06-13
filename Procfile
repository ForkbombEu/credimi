# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
UI: ./scripts/wait-for-it.sh -s -t 0 localhost:8090 && bun run /app/webapp/build/index.js
API: credimi serve --http=0.0.0.0:8090
BE: temporal server start-dev --db-filename /app/pb_data/temporal.db