// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { PocketbaseQueryAgent } from '@/pocketbase/query/index.js';

export const load = async ({ params, fetch }) => {
	const organization = await new PocketbaseQueryAgent(
		{
			collection: 'organizations',
			expand: ['credential_issuers_via_owner', 'wallets_via_owner', 'verifiers_via_owner']
		},
		{ fetch }
	).getOne(params.organization_id);

	return { organization };
};
