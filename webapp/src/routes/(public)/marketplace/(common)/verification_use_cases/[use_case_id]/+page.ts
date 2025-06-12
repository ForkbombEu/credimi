// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';

export const load = async ({ params }) => {
	const verificationUseCase = await new PocketbaseQueryAgent({
		collection: 'verification_use_cases',
		expand: ['verifier']
	}).getOne(params.use_case_id);

	const verifierMarketplaceItem = await pb
		.collection('marketplace_items')
		.getOne(verificationUseCase.verifier);

	return {
		verificationUseCase,
		verifierMarketplaceItem
	};
};
