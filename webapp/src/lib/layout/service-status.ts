// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type ServiceStatus = 'up' | 'degraded' | 'unknown';

/** Uptime Kuma heartbeat status: 0 down, 1 up, 2 pending, 3 maintenance. */
type Heartbeats = Record<string, { status: number }[]>;

/** Overall state of a status page, from the latest heartbeat of each of its monitors. */
export function overallStatus(heartbeatList: Heartbeats | undefined): ServiceStatus {
	const latest = Object.values(heartbeatList ?? {}).map((beats) => beats.at(-1)?.status);
	if (!latest.length || latest.includes(undefined)) return 'unknown';
	return latest.every((status) => status === 1 || status === 3) ? 'up' : 'degraded';
}
