// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { String } from 'effect';

import { pb } from '@/pocketbase';

import type { MarketplaceItem } from '../_utils/utils.js';

export const load = async ({ params, fetch }) => {
	const canonifiedName = Object.values(params)
		.filter((p) => String.isNonEmpty(p))
		.at(0);
	// TODO - Redirect to marketplace filter with the collection filter
	if (!canonifiedName) throw error(500);

	const marketplaceItem = (await pb
		.collection('marketplace_items')
		.getFirstListItem(`canonified_name = '${canonifiedName}'`, { fetch })) as MarketplaceItem;

	return {
		marketplaceItem
	};
};
