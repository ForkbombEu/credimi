// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod/v3';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { HubItemsResponse } from '@/pocketbase/types';

//

export const hubItemTypes = [
	'wallets',
	'credential_issuers',
	'credentials',
	'verifiers',
	'use_cases_verifications',
	'custom_checks',
	'pipelines'
] as const satisfies CollectionName[];

export const hubItemTypeSchema = z.enum(hubItemTypes);

export type HubItemType = z.infer<typeof hubItemTypeSchema>;

//

export interface HubItem extends HubItemsResponse {
	id: string;
	type: HubItemType;
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
// This type is needed as the HubItem type coming from codegen is not good.
// Since `hub_items` is a view collection, that merges multiple collections,
// pocketbase says that each field is of type `json` and not the actual type.
