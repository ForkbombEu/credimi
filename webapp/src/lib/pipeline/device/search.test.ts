// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { DeviceRecord } from './types';

import { filterDevices } from './search';

const RUNNERS: DeviceRecord[] = [
	{
		name: 'Alpha Device',
		path: 'org-a/alpha',
		isOwned: true,
		isPublished: true,
		isOnline: true
	},
	{
		name: 'Beta',
		path: 'org-b/beta-device',
		isOwned: false,
		isPublished: true,
		isOnline: false
	}
];

describe('filterDevices', () => {
	it('returns all devices when search is empty', () => {
		expect(filterDevices(RUNNERS, '')).toHaveLength(2);
	});

	it('filters by name case-insensitively', () => {
		expect(filterDevices(RUNNERS, 'alpha')).toEqual([RUNNERS[0]]);
	});

	it('filters by path case-insensitively', () => {
		expect(filterDevices(RUNNERS, 'beta-device')).toEqual([RUNNERS[1]]);
	});
});
