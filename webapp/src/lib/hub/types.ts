// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod/v3';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { MarketplaceItemsResponse } from '@/pocketbase/types';

//

export const marketplaceItemTypes = [
	'wallets',
	'credential_issuers',
	'credentials',
	'verifiers',
	'use_cases_verifications',
	'custom_checks',
	'pipelines'
] as const satisfies CollectionName[];

export const marketplaceItemTypeSchema = z.enum(marketplaceItemTypes);

export type MarketplaceItemType = z.infer<typeof marketplaceItemTypeSchema>;

//

export interface MarketplaceItem extends MarketplaceItemsResponse {
	id: string;
	type: MarketplaceItemType;
	name: string;
	description: string | null;
	updated: string;
	avatar_file: string | null;
	avatar_url: string | null;
	organization_id: string;
	children: { id: string; name: string; canonified_name: string }[] | null;
	canonified_name: string;
	organization_name: string;
	organization_canonified_name: string;
	path: string;
}
// This type is needed as the MarketplaceItem type coming from codegen is not good.
// Since `marketplace_items` is a view collection, that merges multiple collections,
// pocketbase says that each field is of type `json` and not the actual type.
