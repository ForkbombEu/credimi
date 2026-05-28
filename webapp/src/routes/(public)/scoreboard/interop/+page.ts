// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InteropMatrixResponse, InteropMode } from '$lib/scoreboard/interop/types';

import { error } from '@sveltejs/kit';

const DEFAULT_MODE: InteropMode = 'wallets_credentials';

function normalizeMode(mode: string | null): InteropMode {
	if (mode === 'wallets_credentials' || mode === 'wallets_issuers') {
		return mode;
	}

	return DEFAULT_MODE;
}

export const load = async ({ fetch, url }) => {
	const mode = normalizeMode(url.searchParams.get('mode'));
	const res = await fetch(`/api/scoreboard/interop?mode=${mode}`);
	if (!res.ok) {
		error(res.status, 'Failed to load interoperability matrix');
	}
	const matrix = (await res.json()) as InteropMatrixResponse;
	return { matrix, mode };
};
