// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { RunnerRecord } from './types';

import {
	applyRefreshFailure,
	applyRefreshSuccess,
	createCatalogState
} from './catalog.svelte';

const SAMPLE: RunnerRecord[] = [
	{
		name: 'R1',
		path: 'org/r1',
		isOwned: true,
		isPublished: true,
		isOnline: true
	}
];

describe('catalog readiness', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('isReady is false until first success', () => {
		const state = createCatalogState();
		expect(state.isReady()).toBe(false);
		applyRefreshSuccess(state, SAMPLE);
		expect(state.isReady()).toBe(true);
	});

	it('keeps snapshot and stays ready after later failure', () => {
		const state = createCatalogState();
		applyRefreshSuccess(state, SAMPLE);
		applyRefreshFailure(state);
		expect(state.isReady()).toBe(true);
		expect(state.read()).toEqual(SAMPLE);
	});
});
