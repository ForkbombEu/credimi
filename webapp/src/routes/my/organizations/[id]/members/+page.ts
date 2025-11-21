// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';

import { pb } from '@/pocketbase';

export const load = async ({ params, fetch }) => {
	if (!browser) return;

	const userAuthorization = await pb
		.collection('orgAuthorizations')
		.getFirstListItem(`user = "${pb.authStore.model!.id}" && organization = "${params.id}"`, {
			fetch,
			requestKey: null
		});

	const userRole = await pb.collection('orgRoles').getOne(userAuthorization.role, { fetch });

	return {
		userAuthorization,
		userRole
	};
};
