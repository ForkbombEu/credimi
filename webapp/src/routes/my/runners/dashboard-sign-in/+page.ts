// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { dashboardHost } from '$lib/runner-dashboard/sign-in';

import { LoginReturnSession, loginReturnPath } from '@/auth/login-return';
import { verifyUser } from '@/auth/verifyUser';
import { deLocalizeUrl, redirect } from '@/i18n';

//

// The page leaves the /my layout (see +layout@.svelte), so its login gate does
// not run: check the user here and come back after login.
export const load = async ({ fetch, url }) => {
	if (!(await verifyUser(fetch))) {
		const requested = loginReturnPath(deLocalizeUrl(url).pathname + url.search);
		if (browser && requested) LoginReturnSession.start({ path: requested });
		redirect('/login');
	}

	const runner = url.searchParams.get('runner') ?? '';
	const returnTo = url.searchParams.get('return_to') ?? '';
	const state = url.searchParams.get('state') ?? '';
	const host = dashboardHost(returnTo);
	return {
		request: runner && host && state ? { runner, returnTo, state } : undefined,
		host
	};
};
