// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export const load = async ({ params, fetch }) => {
	const wallet = await pb.collection('wallets').getOne(params.wallet_id, { fetch });
	return {
		wallet
	};
};
