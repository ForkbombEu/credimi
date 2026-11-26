// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { json } from '@sveltejs/kit';
import { overallStatus, type ServiceStatus } from '$lib/layout/service-status';

// The status page behind https://status.credimi.io/.
const HEARTBEATS = 'https://uptime.dyne.org/api/status-page/heartbeat/credimi';
const TTL = 60_000;

let cached: { status: ServiceStatus; at: number } | undefined;

export async function GET({ fetch }) {
	if (!cached || Date.now() - cached.at > TTL) {
		const status = await fetch(HEARTBEATS, { signal: AbortSignal.timeout(4000) })
			.then((response) => response.json())
			.then((body) => overallStatus(body.heartbeatList))
			.catch((): ServiceStatus => 'unknown');
		cached = { status, at: Date.now() };
	}

	return json(cached.status, { headers: { 'cache-control': 'public, max-age=60' } });
}
