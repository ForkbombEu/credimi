// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$marketplace/_utils';

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase/index.js';

import { getCredentialIssuersDetails } from './_partials/credential-issuer-page.svelte';
import { getCredentialsDetails } from './_partials/credential-page.svelte';
import { getUseCaseVerificationDetails } from './_partials/use-case-verification-page.svelte';
import { getVerifierDetails } from './_partials/verifier-page.svelte';
import { getWalletDetails } from './_partials/wallet-page.svelte';

//

export const load = async ({ params, fetch }) => {
	const { type, organization, item } = parsePath(params.path);

	const marketplaceItem = (await pb
		.collection('marketplace_items')
		.getFirstListItem(
			`type = '${type}' && organization_canonified_name = '${organization}' && canonified_name = '${item}'`,
			{ fetch }
		)) as MarketplaceItem;

	const pageDetails = await getPageDetails(marketplaceItem, fetch);

	return {
		marketplaceItem,
		pageDetails
	};
};

function parsePath(path: string) {
	const chunks = path.split('/');
	if (chunks.length !== 3) error(404);
	return {
		type: chunks[0],
		organization: chunks[1],
		item: chunks[2]
	};
}

function getPageDetails(item: MarketplaceItem, fetchFn = fetch) {
	switch (item.type) {
		case 'credential_issuers':
			return getCredentialIssuersDetails(item.id, fetchFn);
		case 'credentials':
			return getCredentialsDetails(item.id, fetchFn);
		case 'wallets':
			return getWalletDetails(item.id, fetchFn);
		case 'verifiers':
			return getVerifierDetails(item.id, fetchFn);
		case 'use_cases_verifications':
			return getUseCaseVerificationDetails(item.id, fetchFn);
		default:
			error(404);
	}
}
