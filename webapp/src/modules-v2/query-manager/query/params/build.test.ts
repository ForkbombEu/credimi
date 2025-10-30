// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe } from 'node:test';
import { expect, it } from 'vitest';

import { build } from './build';

//

describe('build', () => {
	it('should build a query params object', () => {
		const queryParams = build<'credential_issuers'>({
			page: 1,
			perPage: 10,
			filter: ['name = "John"', 'age > 18'],
			sort: ['name', 'ASC'],
			search: 'John',
			searchFields: ['name', 'url'],
			excludeIDs: ['1', '2']
		});

		expect(queryParams).toEqual({
			page: 1,
			perPage: 10,
			filter: '((name = "John") && (age > 18)) && (name ~ "John" || url ~ "John") && (id != "1" && id != "2")',
			sort: '+name'
		});
	});

	it('should build a query params object with default page and perPage', () => {
		const queryParams = build<'credential_issuers'>({
			filter: ['name = "John"', 'age > 18'],
			sort: ['name', 'ASC'],
			excludeIDs: ['1']
		});

		expect(queryParams).toEqual({
			page: 1,
			perPage: 25,
			filter: '((name = "John") && (age > 18)) && (id != "1")',
			sort: '+name'
		});
	});
});
