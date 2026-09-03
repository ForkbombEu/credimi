// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { DeviceRecord } from './types';

export type CatalogSnapshot = {
	ready: boolean;
	devices: DeviceRecord[];
};

export function onRefreshSuccess(_prev: CatalogSnapshot, next: DeviceRecord[]): CatalogSnapshot {
	return { ready: true, devices: next };
}

export function onRefreshFailure(prev: CatalogSnapshot): CatalogSnapshot {
	if (!prev.ready) {
		return { ready: false, devices: [] };
	}

	return prev;
}
