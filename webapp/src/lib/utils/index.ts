// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { invalidateAll } from '$app/navigation';
import { userOrganization } from '$lib/app-state';
import { onMount } from 'svelte';
import { z } from 'zod';

import { verifyUser } from '@/auth/verifyUser';
import { loadFeatureFlags } from '@/features';
import { redirect } from '@/i18n';
import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent } from '@/pocketbase/query';

//

export { loading, runWithLoading } from '$lib/layout/global-loading.svelte';
export * from './schemas';

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

export async function getUserOrganization(options = { fetch }) {
	const organizationAuth = await new PocketbaseQueryAgent(
		{
			collection: 'orgAuthorizations',
			expand: ['organization'],
			filter: `user.id = "${pb.authStore.record?.id}"`
		},
		{ fetch: options.fetch }
	).getFullList();

	const org = organizationAuth.at(0)?.expand?.organization;
	if (browser) userOrganization.current = org;

	return org;
}

//

//

export function setupPollingWithInvalidation(intervalMs: number) {
	onMount(() => {
		const interval = setInterval(() => {
			invalidateAll();
		}, intervalMs);

		return () => {
			clearInterval(interval);
		};
	});
}

const deeplinkGenerationResponseSchema = z.object({
	deeplink: z.string(),
	steps: z.array(z.unknown()),
	output: z.array(z.unknown())
});

export async function generateDeeplinkFromYaml(yaml: string) {
	const res = await pb.send('api/get-deeplink', {
		method: 'POST',
		body: {
			yaml
		},
		requestKey: null
	});
	return deeplinkGenerationResponseSchema.parse(res);
}
