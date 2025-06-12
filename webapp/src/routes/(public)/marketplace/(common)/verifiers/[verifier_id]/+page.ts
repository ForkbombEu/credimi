// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
import { Effect as _, pipe } from 'effect';

export const load = async ({ params }) => {
	const verifier = await new PocketbaseQueryAgent({
		collection: 'verifiers',
		expand: ['credentials', 'verification_use_cases_via_verifier']
	}).getOne(params.verifier_id);

	const [, marketplaceCredentials] = await pipe(
		verifier.credentials,
		_.partition((c) => _.tryPromise(() => pb.collection('marketplace_items').getOne(c))),
		_.runPromise
	);

	const [, marketplaceVerificationUseCases] = await pipe(
		verifier.expand?.verification_use_cases_via_verifier ?? [],
		_.partition((v) => _.tryPromise(() => pb.collection('marketplace_items').getOne(v.id))),
		_.runPromise
	);

	return {
		verifier,
		marketplaceCredentials,
		marketplaceVerificationUseCases
	};
};
