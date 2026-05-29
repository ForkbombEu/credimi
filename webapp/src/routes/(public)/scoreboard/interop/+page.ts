// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { DEFAULT_INTEROP_PAIR } from '$lib/scoreboard/interop/featured-pairs';
import { isInteropHubCollection } from '$lib/scoreboard/interop/interop-hub-collections';
import type { InteropMatrixResponse } from '$lib/scoreboard/interop/types';

import { error, redirect } from '@sveltejs/kit';

export const load = async ({ fetch, url }) => {
	const row = url.searchParams.get('row');
	const column = url.searchParams.get('column');

	if (!row || !column) {
		redirect(
			302,
			`/scoreboard/interop?row=${DEFAULT_INTEROP_PAIR.row}&column=${DEFAULT_INTEROP_PAIR.column}`
		);
	}
	if (!isInteropHubCollection(row) || !isInteropHubCollection(column)) {
		error(400, 'Invalid interoperability matrix axes');
	}

	const res = await fetch(`/api/scoreboard/interop?row=${row}&column=${column}`);
	if (!res.ok) {
		error(res.status, 'Failed to load interoperability matrix');
	}
	const matrix = (await res.json()) as InteropMatrixResponse;
	return { matrix, row, column };
};
