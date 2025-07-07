// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { loadFeatureFlags } from '@/features';
import { verifyUser } from '@/auth/verifyUser';
import { redirect } from '@/i18n';

//

export async function checkAuthFlagAndUser(options: {
	fetch?: typeof fetch;
	onAuthError?: () => void;
	onUserError?: () => void;
}) {
	const {
		fetch: fetchFn = fetch,
		onAuthError = () => {
			error(404);
		},
		onUserError = () => {
			redirect('/login');
		}
	} = options;

	const featureFlags = await loadFeatureFlags(fetchFn);
	if (!featureFlags.AUTH) onAuthError();
	if (!(await verifyUser(fetchFn))) onUserError();
}
