// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InteropMatrixResponse } from '$lib/scoreboard/interop/types';

import { error } from '@sveltejs/kit';

export const load = async ({ fetch }) => {
	const res = await fetch('/api/scoreboard/interop?mode=wallets_issuers');
	if (!res.ok) {
		error(res.status, 'Failed to load interoperability matrix');
	}
	const matrix = (await res.json()) as InteropMatrixResponse;
	return { matrix };
};
