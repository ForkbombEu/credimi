// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';
import type { JsonLd, PageSeo } from '$lib/seo';

import { error } from '@sveltejs/kit';
import { getHubItemTypeEntityData } from '$lib/hub/utils';
import { ogImages, plainText } from '$lib/seo';

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
		pageDetails,
		seo: getHubItemSeo(hubItem)
	};
};

/** Wallets are apps; issuers, verifiers and use cases are services an organization runs. */
const schemaTypes: Partial<Record<HubItem['type'], string>> = {
	wallets: 'SoftwareApplication',
	credential_issuers: 'Service',
	verifiers: 'Service',
	use_cases_verifications: 'Service'
};

function getHubItemSeo(item: HubItem): PageSeo {
	const typeLabel = getHubItemTypeEntityData(item.type).labels.singular;
	const name = item.name ?? item.canonified_name ?? typeLabel;
	const description =
		plainText(item.description) ||
		`${name}: ${typeLabel.toLowerCase()} by ${item.organization_name}, published on the Credimi Hub with a live integration for EUDI conformance and interoperability testing.`;
	const thing: JsonLd = {
		'@context': 'https://schema.org',
		'@type': schemaTypes[item.type] ?? 'CreativeWork',
		name,
		description: plainText(item.description, 500) || description,
		provider: { '@type': 'Organization', name: item.organization_name }
	};

	return { title: `${name} (${typeLabel})`, description, image: ogImages.hub, jsonLd: [thing] };
}

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
