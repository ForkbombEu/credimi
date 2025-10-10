// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

import type { MarketplaceItem } from '../_utils/utils.js';

import { getRestParams } from './_utils';

export const load = async ({ params, fetch, url }) => {
	const { entity, organization } = getRestParams(params.rest ?? '');

	// TODO - Fix this ugly patch
	const type = url.pathname
		.split('marketplace/')
		.at(1)
		?.split(params.rest ?? '')
		.at(0)
		?.replace('/', '');

	const marketplaceItem = (await pb
		.collection('marketplace_items')
		.getFirstListItem(
			`type = '${type}' && canonified_name = '${entity}' && organization_canonified_name = '${organization}'`,
			{ fetch }
		)) as MarketplaceItem;

	return {
		marketplaceItem
	};
};
