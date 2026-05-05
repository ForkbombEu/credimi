// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

//

type GetRecordByCanonifiedPathResult<T = unknown> = { message: string; record?: T };
type CanonifiedRecord = { __canonified_path__?: string };

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
			return attachCanonifiedPath(result.record, path);
		} else {
			return new Error(result.message);
		}
	} catch {
		return new Error('Failed to get record by path');
	}
}

function attachCanonifiedPath<T>(record: T, path: string): T {
	if (!record || typeof record !== 'object' || Array.isArray(record)) {
		return record;
	}

	const canonifiedRecord = record as T & CanonifiedRecord;
	if (!canonifiedRecord.__canonified_path__) {
		canonifiedRecord.__canonified_path__ = path.replace(/^\/+|\/+$/g, '');
	}

	return canonifiedRecord;
}
