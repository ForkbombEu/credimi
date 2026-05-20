// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RunnerRecord } from './types';

export type CatalogSnapshot = {
	ready: boolean;
	runners: RunnerRecord[];
};

export function onRefreshSuccess(_prev: CatalogSnapshot, next: RunnerRecord[]): CatalogSnapshot {
	return { ready: true, runners: next };
}

export function onRefreshFailure(prev: CatalogSnapshot): CatalogSnapshot {
	if (!prev.ready) {
		return { ready: false, runners: [] };
	}

	return prev;
}
