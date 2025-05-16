// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
import { error } from '@sveltejs/kit';

export const load = async ({ fetch }) => {
	const organizationAuth = await new PocketbaseQueryAgent(
		{
			collection: 'orgAuthorizations',
			expand: ['organization'],
			filter: `user.id = "${pb.authStore.record?.id}"`
		},
		{ fetch }
	).getFullList();

	const organization = organizationAuth.at(0)?.expand?.organization;
	if (!organization) error(404, { message: 'Organization not found' });

	const isOrganizationNotEdited = organization.created === organization.updated;

	return {
		organization,
		isOrganizationNotEdited
	};
};
