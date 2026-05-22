// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { parseSelectorResponse } from './query';

describe('parseSelectorResponse', () => {
	it('maps snake_case API body to RunnerRecord', () => {
		const records = parseSelectorResponse({
			runners: [
				{
					name: 'Online owned',
					path: 'usera-s-organization/owned-online',
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
				path: 'usera-s-organization/owned-online',
				description: 'desc',
				isOwned: true,
				isPublished: false,
				isOnline: true
			}
		]);
	});
});
