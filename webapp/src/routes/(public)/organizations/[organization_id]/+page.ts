// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';

export const load = async ({ params, fetch }) => {
	const organization = await pb
		.collection('organizations')
		.getFirstListItem(`canonified_name = "${params.organization_id}"`, {
			fetch
		});

	const marketplaceItems = await pb.collection('marketplace_items').getFullList({
		filter: `organization_id = '${organization.id}'`,
		fetch
	});

	return { organization, marketplaceItems };
};
