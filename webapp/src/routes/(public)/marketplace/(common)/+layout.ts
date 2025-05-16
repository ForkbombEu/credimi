// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { error } from '@sveltejs/kit';
import { String } from 'effect';
import type { MarketplaceItem } from '../_utils/utils.js';

export const load = async ({ params }) => {
	const id = Object.values(params)
		.filter((p) => String.isNonEmpty(p))
		.at(0);
	if (!id) throw error(500);

	const marketplaceItem = (await pb
		.collection('marketplace_items')
		.getOne(id)) as MarketplaceItem;

	return {
		marketplaceItem
	};
};
