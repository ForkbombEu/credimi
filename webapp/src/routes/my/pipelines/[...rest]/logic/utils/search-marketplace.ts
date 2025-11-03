// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem, MarketplaceItemType } from '$lib/marketplace';

import { pb } from '@/pocketbase';

//

export async function searchMarketplace(text: string, type: MarketplaceItemType) {
	const result = await pb.collection('marketplace_items').getFullList({
		filter: `path ~ "${text}" && type = "${type}"`,
		requestKey: null
	});
	return result as MarketplaceItem[];
}
