// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

import type { MarketplaceItem } from '../_utils/utils.js';

import { getRestParams } from './_utils';

export const load = async ({ params, fetch }) => {
	const { entity, organization } = getRestParams(params.rest ?? '');

	const marketplaceItem = (await pb
		.collection('marketplace_items')
		.getFirstListItem(
			`canonified_name = '${entity}' && organization_canonified_name = '${organization}'`,
			{ fetch }
		)) as MarketplaceItem;

	return {
		marketplaceItem
	};
};
