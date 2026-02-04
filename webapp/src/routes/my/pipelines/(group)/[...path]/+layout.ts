// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getRecordByCanonifiedPath } from '$lib/canonify';

import type { PipelinesResponse } from '@/pocketbase/types/index.generated';

//

export const load = async ({ params, fetch }) => {
	const pipeline = await getRecordByCanonifiedPath<PipelinesResponse>(params.path, {
		fetch
	});
	if (pipeline instanceof Error) {
		error(404);
	}
	return { pipeline };
};
