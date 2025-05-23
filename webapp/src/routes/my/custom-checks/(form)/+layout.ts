// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { Either } from 'effect';
import { getStandardsAndTestSuites } from '../../tests/new/_partials/standards-response-schema';
import { checkAuthFlagAndUser } from '$lib/utils';

export const load = async ({ fetch }) => {
	await checkAuthFlagAndUser({ fetch });

	const result = await getStandardsAndTestSuites({ fetch });

	if (Either.isLeft(result)) {
		error(500, { message: result.left.message });
	} else {
		return {
			standardsAndTestSuites: result.right
		};
	}
};
