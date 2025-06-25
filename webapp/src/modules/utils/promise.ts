// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Effect as _, pipe } from 'effect';

/**
 * Partitions an array of promises into successes and failures.
 *
 * @param promises - The promises to partition.
 * @returns A tuple of successes and failures.
 */
export async function partitionPromises<T>(promises: Promise<T>[]): Promise<[T[], Error[]]> {
	const [failures, successes] = await pipe(
		promises,
		_.partition((p) => _.tryPromise(() => p)),
		_.runPromise
	);

	return [successes, failures];
}
