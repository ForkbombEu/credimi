// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase';

export const load = async ({ fetch, parent }) => {
	const { organization } = await parent();
	if (!organization) throw error(404);

	const hubItems = await pb.collection('hub_items').getFullList({
		filter: `organization_id = '${organization.id}'`,
		fetch
	});

	const isOrganizationNotEdited = organization.created === organization.updated;

	return { organization, hubItems, isOrganizationNotEdited };
};
