// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query';

//

export const load = async ({ params }) => {
	const credentialIssuer = await new PocketbaseQueryAgent(
		{
			collection: 'credential_issuers',
			expand: ['credentials_via_credential_issuer']
		},
		{ fetch }
	).getOne(params.issuer_id);

	const credentialsIds = (credentialIssuer.expand?.credentials_via_credential_issuer ?? []).map(
		(credential) => credential.id
	);

	const credentialsFilters = credentialsIds.map((id) => `id = '${id}'`).join(' || ');

	const credentialsMarketplaceItems = await pb.collection('marketplace_items').getFullList(1, {
		filter: credentialsFilters
	});

	return {
		credentialIssuer,
		credentialsMarketplaceItems
	};
};
