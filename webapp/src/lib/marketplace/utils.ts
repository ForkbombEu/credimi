// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { CheckCheck, CheckCircle, QrCode, ShieldCheck, Users, Wallet } from 'lucide-svelte';
import { z } from 'zod';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { MarketplaceItemsResponse } from '@/pocketbase/types';

import { localizeHref, m } from '@/i18n';
import { pb } from '@/pocketbase';

import MarketplaceItemCard from './marketplace-item-card.svelte';
import MarketplaceItemTypeDisplay from './marketplace-item-type-display.svelte';

//

export { MarketplaceItemCard, MarketplaceItemTypeDisplay };

/* -- Marketplace item types -- */

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

/* -- Marketplace item type -- */

// This type is needed as the MarketplaceItem type coming from codegen is not good.
// Since `marketplace_items` is a view collection, that merges multiple collections,
// pocketbase says that each field is of type `json` and not the actual type.

export interface MarketplaceItem {
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

/* -- Marketplace item type mapping to display data -- */

export type MarketplaceItemDisplayData = {
	label: string;
	labelPlural: string;
	bgClass: string;
	textClass: string;
	backgroundClass: string;
	outlineClass: string;
	icon: typeof Wallet;
};

type MarketplaceItemsDisplayConfig = Record<MarketplaceItemType, MarketplaceItemDisplayData>;

export const marketplaceItemsDisplayConfig = {
	wallets: {
		label: m.Wallet(),
		labelPlural: m.Wallets(),
		bgClass: 'bg-[hsl(var(--blue-foreground))]',
		textClass: 'text-[hsl(var(--blue-foreground))]',
		backgroundClass: 'bg-[hsl(var(--blue-background))]',
		outlineClass: 'border-[hsl(var(--blue-outline))]',
		icon: Wallet
	},
	custom_checks: {
		label: m.Custom_check(),
		labelPlural: m.Custom_checks(),
		bgClass: 'bg-[hsl(var(--purple-foreground))]',
		textClass: 'text-[hsl(var(--purple-foreground))]',
		backgroundClass: 'bg-[hsl(var(--purple-background))]',
		outlineClass: 'border-[hsl(var(--purple-outline))]',
		icon: CheckCheck
	},
	credential_issuers: {
		label: m.Issuer(),
		labelPlural: m.Issuers(),
		bgClass: 'bg-[hsl(var(--green-foreground))]',
		textClass: 'text-[hsl(var(--green-foreground))]',
		backgroundClass: 'bg-[hsl(var(--green-background))]',
		outlineClass: 'border-[hsl(var(--green-outline))]',
		icon: Users
	},
	credentials: {
		label: m.Credential(),
		labelPlural: m.Credentials(),
		bgClass: 'bg-[hsl(var(--green-light-foreground))]',
		textClass: 'text-[hsl(var(--green-light-foreground))]',
		backgroundClass: 'bg-[hsl(var(--green-light-background))]',
		outlineClass: 'border-[hsl(var(--green-light-outline))]',
		icon: QrCode
	},
	verifiers: {
		label: m.Verifier(),
		labelPlural: m.Verifiers(),
		bgClass: 'bg-[hsl(var(--red-foreground))]',
		textClass: 'text-[hsl(var(--red-foreground))]',
		backgroundClass: 'bg-[hsl(var(--red-background))]',
		outlineClass: 'border-[hsl(var(--red-outline))]',
		icon: ShieldCheck
	},
	use_cases_verifications: {
		label: m.Use_case_verification(),
		labelPlural: m.Use_case_verifications(),
		bgClass: 'bg-[hsl(var(--orange-foreground))]',
		textClass: 'text-[hsl(var(--orange-foreground))]',
		backgroundClass: 'bg-[hsl(var(--orange-background))]',
		outlineClass: 'border-[hsl(var(--orange-outline))]',
		icon: CheckCircle
	},
	pipelines: {
		label: m.Pipelines(),
		labelPlural: m.Pipelines(),
		bgClass: 'bg-orange-600',
		textClass: 'text-orange-600',
		backgroundClass: 'bg-orange-100',
		outlineClass: 'border-orange-600',
		icon: CheckCircle
	}
} satisfies MarketplaceItemsDisplayConfig;

export function getMarketplaceItemTypeData(type: MarketplaceItemType) {
	const display = marketplaceItemsDisplayConfig[type];
	const filter = `type = '${type}'`;
	return { display, filter };
}

export const CUSTOM_CHECK_QUERY_PARAM = 'custom_check_id';

export function getMarketplaceItemUrl(item: MarketplaceItem) {
	const href =
		item.type === 'custom_checks'
			? `/my/tests/new?${CUSTOM_CHECK_QUERY_PARAM}=${item.id}`
			: `/marketplace/${item.path}`;
	return localizeHref(href);
}

export function getMarketplaceItemData(item: MarketplaceItem) {
	const href = getMarketplaceItemUrl(item);

	const logo = item.avatar_file
		? pb.files.getURL(
				item.type === 'pipelines'
					? { collectionName: 'organizations', id: item.organization_id }
					: { collectionName: item.type, id: item.id },
				item.avatar_file
			)
		: item.avatar_url
			? item.avatar_url
			: undefined;

	return { href, logo, ...getMarketplaceItemTypeData(item.type) };
}

//

export function isCredentialIssuer(item: MarketplaceItemsResponse): boolean {
	return item.type === marketplaceItemTypes[1];
}

export function isVerifier(item: MarketplaceItemsResponse): boolean {
	return item.type === marketplaceItemTypes[3];
}
