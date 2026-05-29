// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { normalizeInteropMode } from '$lib/scoreboard/interop/modes';
import type { InteropMatrixResponse } from '$lib/scoreboard/interop/types';

import { error } from '@sveltejs/kit';

export const load = async ({ fetch, url }) => {
	const mode = normalizeInteropMode(url.searchParams.get('mode'));
	const res = await fetch(`/api/scoreboard/interop?mode=${mode}`);
	if (!res.ok) {
		error(res.status, 'Failed to load interoperability matrix');
	}
	const matrix = (await res.json()) as InteropMatrixResponse;
	return { matrix, mode };
};
