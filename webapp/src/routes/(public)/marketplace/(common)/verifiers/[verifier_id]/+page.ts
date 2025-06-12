// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
import { Effect as _, pipe } from 'effect';

export const load = async ({ params }) => {
	const verifier = await new PocketbaseQueryAgent({
		collection: 'verifiers',
		expand: ['use_cases_verifications_via_verifier']
	}).getOne(params.verifier_id);

	const useCasesVerifications = verifier.expand?.use_cases_verifications_via_verifier ?? [];

	const [, marketplaceUseCasesVerifications] = await pipe(
		useCasesVerifications,
		_.partition((v) => _.tryPromise(() => pb.collection('marketplace_items').getOne(v.id))),
		_.runPromise
	);

	const [, marketplaceCredentials] = await pipe(
		useCasesVerifications.flatMap((v) => v.credentials),
		_.partition((c) => _.tryPromise(() => pb.collection('marketplace_items').getOne(c))),
		_.runPromise
	);

	return {
		verifier,
		marketplaceCredentials,
		marketplaceUseCasesVerifications
	};
};
