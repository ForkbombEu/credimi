// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export const load = async ({ params, fetch }) => {
	const wallet = await pb
		.collection('wallets')
		.getFirstListItem(`canonified_name = '${params.wallet_name}'`, { fetch });

	const actions = await pb.collection('wallet_actions').getFullList({
		filter: `wallet="${wallet.id}"`,
		fetch
	});

	return {
		wallet,
		actions
	};
};
