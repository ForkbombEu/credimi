// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { blockNonMembers } from '@/organizations';

export const load = async ({ params, fetch }) => {
	const organizationId = params.id;

	await blockNonMembers(organizationId, fetch);

	const organization = await pb.collection('organizations').getOne(organizationId, {
		fetch,
		requestKey: null
	});

	return { organization };
};
