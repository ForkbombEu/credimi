// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InteropMatrixResponse, InteropMode } from '$lib/scoreboard/interop/types';

import { error } from '@sveltejs/kit';

const SUPPORTED_MODES: InteropMode[] = [
	'wallets_credentials',
	'wallets_issuers',
	'wallets_verifiers',
	'wallets_use_case_verifications'
];

const DEFAULT_MODE: InteropMode = 'wallets_credentials';

function normalizeMode(mode: string | null): InteropMode {
	if (SUPPORTED_MODES.includes(mode as InteropMode)) {
		return mode as InteropMode;
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
