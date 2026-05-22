// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { RunnerRecord } from './types';

import { filterRunners } from './search';

const RUNNERS: RunnerRecord[] = [
	{
		name: 'Alpha Runner',
		path: 'org-a/alpha',
		isOwned: true,
		isPublished: true,
		isOnline: true
	},
	{
		name: 'Beta',
		path: 'org-b/beta-runner',
		isOwned: false,
		isPublished: true,
		isOnline: false
	}
];

describe('filterRunners', () => {
	it('returns all runners when search is empty', () => {
		expect(filterRunners(RUNNERS, '')).toHaveLength(2);
	});

	it('filters by name case-insensitively', () => {
		expect(filterRunners(RUNNERS, 'alpha')).toEqual([RUNNERS[0]]);
	});

	it('filters by path case-insensitively', () => {
		expect(filterRunners(RUNNERS, 'beta-runner')).toEqual([RUNNERS[1]]);
	});
});
