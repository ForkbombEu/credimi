// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';

export const load = async ({ params }) => {
	const verifier = await new PocketbaseQueryAgent({
		collection: 'verifiers',
		expand: ['credentials']
	}).getOne(params.verifier_id);

	const credentials = (verifier.expand?.credentials ?? []).filter((c) => c.published);

	return {
		verifier,
		credentials
	};
};
