// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getStandardsWithTestSuites } from '$lib/standards';
import { checkAuthFlagAndUser } from '$lib/utils';

export const load = async ({ fetch }) => {
	await checkAuthFlagAndUser({ fetch });

	const result = await getStandardsWithTestSuites({ fetch });

	if (result instanceof Error) {
		error(500, { message: result.message });
	} else {
		return {
			standardsAndTestSuites: result
		};
	}
};
