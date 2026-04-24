// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { Collections } from '@/pocketbase/types';

export const load = async ({ fetch }) => {
	const wallets = await pb.collection('marketplace_items').getList(1, 3, {
		filter: `type = '${Collections.Wallets}'`,
		fetch,
		sort: '@random'
	});
	const issuers = await pb.collection('marketplace_items').getList(1, 3, {
		filter: `type = '${Collections.CredentialIssuers}'`,
		fetch,
		sort: '@random'
	});
	const verifiers = await pb.collection('marketplace_items').getList(1, 3, {
		filter: `type = '${Collections.Verifiers}'`,
		fetch,
		sort: '@random'
	});

	return {
		wallets: wallets.items,
		issuers: issuers.items,
		verifiers: verifiers.items
	};
};
