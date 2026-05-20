// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { onRefreshFailure, onRefreshSuccess, type CatalogSnapshot } from './catalog-state';
import type { RunnerRecord } from './types';

const SAMPLE: RunnerRecord[] = [
	{
		name: 'R1',
		path: 'org/r1',
		isOwned: true,
		isPublished: true,
		isOnline: true
	}
];

function emptySnapshot(): CatalogSnapshot {
	return { ready: false, runners: [] };
}

describe('catalog readiness', () => {
	it('isReady is false until first success', () => {
		expect(emptySnapshot().ready).toBe(false);
		expect(onRefreshSuccess(emptySnapshot(), SAMPLE).ready).toBe(true);
	});

	it('keeps snapshot and stays ready after later failure', () => {
		const afterSuccess = onRefreshSuccess(emptySnapshot(), SAMPLE);
		const afterFailure = onRefreshFailure(afterSuccess);

		expect(afterFailure.ready).toBe(true);
		expect(afterFailure.runners).toEqual(SAMPLE);
	});

	it('clears runners when refresh fails before first success', () => {
		expect(onRefreshFailure(emptySnapshot())).toEqual({ ready: false, runners: [] });
	});
});
