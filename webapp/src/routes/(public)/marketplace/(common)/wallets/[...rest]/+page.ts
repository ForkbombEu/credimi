// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

import { getFilterFromRestParams } from '../../_utils';

export const load = async ({ params, fetch }) => {
	const filter = getFilterFromRestParams(params.rest);
	const wallet = await pb.collection('wallets').getFirstListItem(filter, { fetch });

	const actions = await pb.collection('wallet_actions').getFullList({
		filter: `wallet="${wallet.id}"`,
		fetch
	});

	return {
		wallet,
		actions
	};
};
