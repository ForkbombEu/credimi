// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

//

export type DashboardSignInRequest = { runner: string; returnTo: string; state: string };

/** The host of the dashboard a sign-in returns to, or undefined for anything but an http(s) URL. */
export function dashboardHost(returnTo: string): string | undefined {
	let url: URL;
	try {
		url = new URL(returnTo);
	} catch {
		return undefined;
	}
	if (url.protocol !== 'http:' && url.protocol !== 'https:') return undefined;
	return url.host || undefined;
}

/** Asks Credimi for a one-time sign-in code and returns the dashboard URL that redeems it. */
export async function requestDashboardCode(request: DashboardSignInRequest): Promise<string> {
	const response = await pb.send<{ redirect: string }>('/api/mobile-runner/dashboard-code', {
		method: 'POST',
		body: { runner_id: request.runner, return_to: request.returnTo, state: request.state },
		requestKey: null
	});
	return response.redirect;
}
