// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { loadFeatureFlags } from '@/features';
import { verifyUser } from '@/auth/verifyUser';
import { redirect } from '@/i18n';
import { z } from 'zod';
import { parse as parseYaml } from 'yaml';
import { getExceptionMessage } from '@/utils/errors';
import { Record as R } from 'effect';
import { PocketbaseQueryAgent } from '@/pocketbase/query';
import { pb } from '@/pocketbase';
import { invalidateAll } from '$app/navigation';
import { onMount } from 'svelte';
import { userOrganization } from '$lib/app-state';
import { browser } from '$app/environment';

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

export const yamlStringSchema = z
	.string()
	.nonempty()
	.superRefine((value, ctx) => {
		try {
			parseYaml(value);
		} catch (e) {
			ctx.addIssue({
				code: z.ZodIssueCode.custom,
				message: `Invalid YAML: ${getExceptionMessage(e)}`
			});
		}
	});

export const jsonStringSchema = z
	.string()
	.nonempty()
	.superRefine((v, ctx) => {
		try {
			z.record(z.string(), z.unknown())
				.refine((value) => R.size(value) > 0)
				.parse(JSON.parse(v));
		} catch (e) {
			const message = getExceptionMessage(e);
			ctx.addIssue({
				code: z.ZodIssueCode.custom,
				message: `Invalid JSON object: ${message}`
			});
		}
	});

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
