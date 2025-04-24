// SPDX-FileCopyrightText: 2025 Forkbomb BV

// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent } from '@/pocketbase/query/index.js';
import { error } from '@sveltejs/kit';

export const load = async ({ params }) => {
	const { collection, record_id } = params;

	try {
		if (collection === 'credential_issuers') {
			return await new PocketbaseQueryAgent(
				{
					collection: 'credential_issuers',
					expand: ['credentials_via_credential_issuer']
				},
				{ fetch }
			).getOne(record_id);
		}
		//
		else if (collection === 'verifiers') {
			return await pb.collection('verifiers').getOne(record_id);
		}
		//
		else {
			return await pb.collection('wallets').getOne(record_id);
		}
	} catch {
		error(404, {
			message: 'Record not found'
		});
	}
};
