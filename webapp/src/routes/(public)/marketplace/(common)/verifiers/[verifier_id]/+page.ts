// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
import { partitionPromises } from '@/utils/promise';

export const load = async ({ params, fetch }) => {
	const verifier = await new PocketbaseQueryAgent(
		{
			collection: 'verifiers',
			expand: ['use_cases_verifications_via_verifier']
		},
		{ fetch }
	).getOne(params.verifier_id);

	const useCasesVerifications = verifier.expand?.use_cases_verifications_via_verifier ?? [];

	const [marketplaceUseCasesVerifications] = await partitionPromises(
		useCasesVerifications.map((v) => pb.collection('marketplace_items').getOne(v.id, { fetch }))
	);

	const [marketplaceCredentials] = await partitionPromises(
		useCasesVerifications
			.flatMap((v) => v.credentials)
			.map((c) => pb.collection('marketplace_items').getOne(c, { fetch }))
	);

	return {
		verifier,
		marketplaceCredentials,
		marketplaceUseCasesVerifications
	};
};
