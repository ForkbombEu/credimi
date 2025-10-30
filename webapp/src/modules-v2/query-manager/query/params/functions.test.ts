// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { QueryParams } from './types';

import { deserialize, merge, serialize } from './functions';

//

describe('utils', () => {
	it('should serialize and deserialize query params', () => {
		const queryParams: QueryParams<never> = {
			page: 1,
			perPage: 10,
			filter: [
				'name = "John"',
				{ id: '1', expressions: ['name = "John"', 'age > 18'], mode: 'AND' }
			],
			sort: [
				['name', 'ASC'],
				['age', 'DESC']
			],
			search: 'John',
			searchFields: ['name', 'email']
		};

		const built = serialize(queryParams);
		const parsed = deserialize(built);

		expect(parsed).toEqual(queryParams);
	});

	it('should merge query params', () => {
		const params1: QueryParams<never> = {
			page: 1,
			perPage: 10,
			filter: [{ id: '1', expressions: ['name = "John"', 'age > 18'], mode: 'AND' }],
			sort: ['name', 'ASC']
		};
		const params2: QueryParams<never> = {
			page: 2,
			perPage: 20,
			filter: ['name = "Jane"'],
			sort: ['age', 'DESC']
		};
		const merged = merge(params1, params2);
		expect(merged).toEqual({
			page: 2,
			perPage: 20,
			filter: [
				{ id: '1', expressions: ['name = "John"', 'age > 18'], mode: 'AND' },
				'name = "Jane"'
			],
			sort: [
				['name', 'ASC'],
				['age', 'DESC']
			]
		});
	});
});
