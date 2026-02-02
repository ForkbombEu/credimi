// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { partitionPromises } from './promise';

describe('partitionPromises', () => {
	it('splits fulfilled and rejected promises', async () => {
		const failure = new Error('nope');
		const [successes, failures] = await partitionPromises([
			Promise.resolve('a'),
			Promise.reject(failure),
			Promise.resolve('b')
		]);

		expect(successes).toEqual(['a', 'b']);
		expect(failures).toHaveLength(1);
		expect(failures[0]).toBeInstanceOf(Error);
	});
});
