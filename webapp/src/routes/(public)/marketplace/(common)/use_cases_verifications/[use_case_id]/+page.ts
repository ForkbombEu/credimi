// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';

export const load = async ({ params }) => {
	const useCaseVerification = await new PocketbaseQueryAgent({
		collection: 'use_cases_verifications',
		expand: ['verifier']
	}).getOne(params.use_case_id);

	const verifierMarketplaceItem = await pb
		.collection('marketplace_items')
		.getOne(useCaseVerification.verifier);

	return {
		useCaseVerification,
		verifierMarketplaceItem
	};
};
