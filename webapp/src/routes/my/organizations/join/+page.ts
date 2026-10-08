// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';

import { pb } from '@/pocketbase';

export const load = async ({ fetch }) => {
	if (!browser) return;

	// Memberships are readable only within one's own organizations, so joinable
	// organizations are found by excluding the caller's own.
	const memberships = await pb.collection('orgAuthorizations').getFullList({
		filter: pb.filter('user = {:user}', { user: pb.authStore.record?.id }),
		fields: 'organization',
		fetch,
		requestKey: null
	});

	return {
		memberOrganizationIds: memberships.map((membership) => membership.organization)
	};
};
