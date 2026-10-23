// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { absoluteUrl, plainText, SITE_ORIGIN } from '$lib/seo';

import { pb } from '@/pocketbase/index.js';

export const load = async ({ params, fetch }) => {
	const organization = await pb
		.collection('organizations')
		.getFirstListItem(`canonified_name = "${params.organization_id}"`, {
			fetch
		});

	const hubItems = await pb.collection('hub_items').getFullList({
		filter: `organization_id = '${organization.id}'`,
		fetch
	});

	const logo = organization.logo ? pb.files.getURL(organization, organization.logo) : undefined;
	const description =
		plainText(organization.description) ||
		`${organization.name} publishes ${hubItems.length} EUDI wallets, issuers, verifiers and pipelines on the Credimi Hub.`;

	return {
		organization,
		hubItems,
		seo: {
			title: organization.name,
			description,
			jsonLd: [
				{
					'@context': 'https://schema.org',
					'@type': 'Organization',
					name: organization.name,
					description: plainText(organization.description, 500) || description,
					url: organization.external_website_url || undefined,
					logo: logo ? absoluteUrl(logo, SITE_ORIGIN) : undefined
				}
			]
		}
	};
};
