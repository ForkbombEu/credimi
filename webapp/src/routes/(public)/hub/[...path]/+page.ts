// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase/index.js';

import { getCredentialIssuersDetails } from './_partials/credential-issuer-page.svelte';
import { getCredentialsDetails } from './_partials/credential-page.svelte';
import { getPipelineDetails } from './_partials/pipeline-page.svelte';
import { getUseCaseVerificationDetails } from './_partials/use-case-verification-page.svelte';
import { getVerifierDetails } from './_partials/verifier-page.svelte';
import { getWalletDetails } from './_partials/wallet-page.svelte';

//

export const load = async ({ params, fetch }) => {
	const fullPath = params.path;

	const hubItem: HubItem = await pb
		.collection('hub_items')
		.getFirstListItem(pb.filter('path = {:path}', { path: fullPath }), {
			fetch
		});
	const pageDetails = await getPageDetails(hubItem, fetch);

	return {
		hubItem,
		pageDetails
	};
};

function getPageDetails(item: HubItem, fetchFn = fetch) {
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
		case 'pipelines':
			return getPipelineDetails(item.id, fetchFn);
		default:
			error(404);
	}
}
