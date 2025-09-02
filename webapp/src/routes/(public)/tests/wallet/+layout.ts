// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { getWalletTestParams } from './_partials';

export const load = ({ url }) => {
	const params = getWalletTestParams(url);

	if (!params.workflowId) {
		error(404);
	}

	return params;
};
