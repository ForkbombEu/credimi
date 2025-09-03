// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { loadFeatureFlags } from '@/features/index.js';
import { redirect } from '@/i18n';
import { OrganizationInviteSession } from '@/organizations/invites';
import { pb } from '@/pocketbase';

export const load = async ({ params, fetch }) => {
	const featureFlags = await loadFeatureFlags(fetch);
	if (!featureFlags.ORGANIZATIONS) error(404);

	OrganizationInviteSession.start({
		organizationId: params.orgId,
		inviteId: params.inviteId,
		email: decodeURIComponent(params.email),
		userId: params.userId
	});

	if (pb.authStore.token) redirect('/my/organizations');
	else {
		if (params.userId) redirect('/login');
		else redirect('/register');
	}
};
