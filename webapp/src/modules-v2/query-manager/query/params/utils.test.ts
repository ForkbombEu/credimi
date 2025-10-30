// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { SortParam } from './types';

import { ensureSortArray } from './utils';

//

describe('utils', () => {
	it('should ensure sort array with single item', () => {
		const sort: SortParam<never> = ['name', 'ASC'];
		const sorted = ensureSortArray(sort);
		expect(sorted).toEqual([['name', 'ASC']]);
	});

	it('should ensure sort array with multiple items', () => {
		const sort: SortParam<never> = [
			['name', 'ASC'],
			['age', 'DESC']
		];
		const sorted = ensureSortArray(sort);
		expect(sorted).toEqual([
			['name', 'ASC'],
			['age', 'DESC']
		]);
	});
});
