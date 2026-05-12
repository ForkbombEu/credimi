// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem, MarketplaceItemType } from '$lib/marketplace';

import { pb } from '@/pocketbase';

//

export async function searchMarketplace(path: string, type: MarketplaceItemType) {
	const result = await pb.collection('marketplace_items').getList(1, 10, {
		filter: pb.filter('path ~ {:path} && type = {:type}', { path, type }),
		requestKey: null
	});
	return result.items as MarketplaceItem[];
}
