// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getRecordByCanonifiedPath } from '$lib/canonify';

import type { CustomChecksResponse } from '@/pocketbase/types';

//

export const load = async ({ params, fetch }) => {
	const path = params.path;

	const record = await getRecordByCanonifiedPath<CustomChecksResponse>(path, { fetch });
	if (record instanceof Error) error(404);

	return { record };
};
