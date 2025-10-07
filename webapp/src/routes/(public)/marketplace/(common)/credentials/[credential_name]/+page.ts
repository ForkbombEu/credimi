// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
import { Collections } from '@/pocketbase/types/index.generated.js';

export const load = async ({ params, fetch }) => {
	try {
		const credential = await new PocketbaseQueryAgent(
			{
				collection: 'credentials',
				expand: ['credential_issuer']
			},
			{ fetch }
		).getFirstListItem(`canonified_name = '${params.credential_name}'`);

		const credentialIssuerMarketplaceEntry = await pb
			.collection('marketplace_items')
			.getFirstListItem(
				`id = '${credential.credential_issuer}' && type = '${Collections.CredentialIssuers}'`,
				{ fetch }
			);

		const credentialIssuer = credential.expand?.credential_issuer;
		if (!credentialIssuer) throw new Error('Credential issuer not found');

		return {
			credential,
			credentialIssuer,
			credentialIssuerMarketplaceEntry
		};
	} catch {
		error(404);
	}
};
