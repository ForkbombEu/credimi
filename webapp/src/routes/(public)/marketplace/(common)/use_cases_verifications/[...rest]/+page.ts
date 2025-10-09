// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
import { partitionPromises } from '@/utils/promise';

import { getFilterFromRestParams } from '../../_utils';

export const load = async ({ params, fetch }) => {
	const filter = getFilterFromRestParams(params.rest);
	const useCaseVerification = await new PocketbaseQueryAgent(
		{
			collection: 'use_cases_verifications',
			expand: ['verifier', 'credentials']
		},
		{ fetch }
	).getFirstListItem(filter);

	const verifierMarketplaceItem = await pb
		.collection('marketplace_items')
		.getOne(useCaseVerification.verifier, { fetch });

	const [marketplaceCredentials] = await partitionPromises(
		useCaseVerification.credentials.map((c) =>
			pb.collection('marketplace_items').getOne(c, { fetch })
		)
	);

	return {
		useCaseVerification,
		verifierMarketplaceItem,
		marketplaceCredentials
	};
};
