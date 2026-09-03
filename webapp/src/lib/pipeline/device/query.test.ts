// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { parseSelectorResponse } from './query';

describe('parseSelectorResponse', () => {
	it('maps snake_case API body to DeviceRecord', () => {
		const records = parseSelectorResponse({
			devices: [
				{
					name: 'Online owned',
					path: 'usera-s-organization/owned-host/device-a',
					runner_id: 'usera-s-organization/owned-host',
					runner_name: 'Owned host',
					description: 'desc',
					is_owned: true,
					is_published: false,
					is_online: true
				}
			]
		});

		expect(records).toEqual([
			{
				name: 'Online owned',
				path: 'usera-s-organization/owned-host/device-a',
				runnerId: 'usera-s-organization/owned-host',
				runnerName: 'Owned host',
				description: 'desc',
				isOwned: true,
				isPublished: false,
				isOnline: true
			}
		]);
	});
});
