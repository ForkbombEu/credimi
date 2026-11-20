// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase';

export const load = async ({ fetch, parent }) => {
	const { organization } = await parent();
	if (!organization) throw error(404);

	const hubItems = await pb.collection('hub_items').getFullList({
		filter: pb.filter('organization_id = {:id}', { id: organization.id }),
		fetch
	});

	const isOrganizationNotEdited = organization.created === organization.updated;

	// Public organizations are the ones listed on the hub.
	const hubListing = await pb.collection('hub_organizations').getList(1, 1, {
		filter: pb.filter('id = {:id}', { id: organization.id }),
		fetch
	});
	const isPublic = hubListing.totalItems > 0;

	return { organization, hubItems, isOrganizationNotEdited, isPublic };
};
