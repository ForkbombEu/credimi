// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { error } from '@sveltejs/kit';

export const load = async ({ fetch, parent }) => {
	const { organization } = await parent();
	if (!organization) throw error(404);

	const marketplaceItems = await pb.collection('marketplace_items').getFullList({
		filter: `organization_id = '${organization.id}'`,
		fetch
	});

	return { organization, marketplaceItems };
};
