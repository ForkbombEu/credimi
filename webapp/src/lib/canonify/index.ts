// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

//

type GetRecordByCanonifiedPathResult<T = unknown> = { message: string; record?: T };

export async function getRecordByCanonifiedPath<T = unknown>(
	path: string,
	options = { fetch }
): Promise<T | Error> {
	try {
		const result = await pb.send<GetRecordByCanonifiedPathResult<T>>(
			'/api/canonify/identifier/validate',
			{
				method: 'POST',
				body: { canonified_name: path },
				fetch: options.fetch,
				requestKey: null
			}
		);
		if (result.record) {
			return result.record;
		} else {
			return new Error(result.message);
		}
	} catch {
		return new Error('Failed to get record by path');
	}
}
