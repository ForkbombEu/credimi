// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { verifyUser } from '@/auth/verifyUser';
import { loadFeatureFlags } from '@/features';
import { error } from '@sveltejs/kit';
import { redirect } from '@/i18n';

import { Either } from 'effect';
import { getStandardsAndTestSuites } from '../../tests/new/_partials/standards-response-schema';

export const load = async ({ fetch }) => {
	const featureFlags = await loadFeatureFlags(fetch);
	if (!featureFlags.AUTH) error(404);
	if (!(await verifyUser(fetch))) redirect('/login');

	const result = await getStandardsAndTestSuites({ fetch });

	if (Either.isLeft(result)) {
		error(500, { message: result.left.message });
	} else {
		return {
			standardsAndTestSuites: result.right
		};
	}
};
