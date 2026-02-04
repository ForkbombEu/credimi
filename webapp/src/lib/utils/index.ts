// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { userOrganization } from '$lib/app-state';
import slugify from 'slugify';
import { z } from 'zod/v3';

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
		{ fetch: options.fetch, requestKey: null }
	).getFullList();

	const org = organizationAuth.at(0)?.expand?.organization;
	if (browser) userOrganization.current = org;

	return org;
}

//

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

//

export function getPath<T extends object>(record: T, trim = false) {
	if ('__canonified_path__' in record) {
		const path = record.__canonified_path__ as string;
		if (trim) return removeLeadingAndTrailingSlashes(path);
		return path;
	}
	return '__no_path__';
}

export function slug(string: string) {
	return slugify(string, {
		replacement: '-',
		remove: /[*+~.()'"!:@]/g,
		lower: true,
		strict: true
	});
}

/**
 * Merges multiple path segments into a single normalized path.
 * Handles slashes, duplicate slashes, and relative segments.
 * Removes leading and trailing slashes.
 *
 * @param paths - Path segments to merge
 * @returns A normalized path string
 *
 * @example
 * mergePaths('/api', 'users', '/123') // '/api/users/123'
 * mergePaths('api/', '/users/', '123') // 'api/users/123'
 * mergePaths('api', '', 'users') // 'api/users'
 */
export function mergePaths(...paths: (string | undefined | null)[]): string {
	const filtered = paths
		.filter((p): p is string => Boolean(p))
		.map((p) => p.trim())
		.filter((p) => p.length > 0);

	return filtered
		.map((p) => {
			if (p.startsWith('/')) p = p.slice(1);
			if (p.endsWith('/')) p = p.slice(0, -1);
			return p;
		})
		.filter((p) => p.length > 0)
		.join('/');
}

export function removeLeadingAndTrailingSlashes(path: string) {
	return path.replace(/^\//, '').replace(/\/$/, '');
}
