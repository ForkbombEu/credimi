// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { loadFeatureFlags } from '@/features';
import { verifyUser } from '@/auth/verifyUser';
import { redirect } from '@/i18n';
import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent } from '@/pocketbase/query';

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

//

export async function getUserOrganization(options = { fetch }) {
	const authorizationQuery = new PocketbaseQueryAgent(
		{
			collection: 'orgAuthorizations',
			filter: `user.id = "${pb.authStore.record?.id}"`,
			expand: ['organization']
		},
		{
			fetch: options.fetch
		}
	);
	const authorization = (await authorizationQuery.getFullList()).at(0);
	if (!authorization || !authorization.expand?.organization) return undefined;

	try {
		const organizationInfo = await pb
			.collection('organization_info')
			.getFirstListItem(`owner.id = "${authorization.organization}"`, {
				fetch: options.fetch
			});

		return { organizationInfo, organization: authorization.expand.organization };
	} catch (e) {
		console.log(e);
		return undefined;
	}
}
