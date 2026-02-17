// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify';

export const load = async ({ params, fetch }) => {
	const path = params.path;

	const record = await getRecordByCanonifiedPath(path, { fetch });
	if (record instanceof Error) {
		throw record;
	}

	return { record };
};
