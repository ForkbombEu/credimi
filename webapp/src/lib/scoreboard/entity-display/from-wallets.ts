// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities } from '$lib/global';

import type { WalletsResponse, WalletVersionsResponse } from '@/pocketbase/types';

import { fromPocketbaseEntity } from './from-pocketbase';
import type { Item } from './types';

//

export type WalletRow = {
	wallet: WalletsResponse;
	version?: WalletVersionsResponse;
};

export function fromWalletRows(rows: WalletRow[]): Item[] {
	return rows.map(({ wallet, version }) => ({
		...fromPocketbaseEntity(wallet, entities.wallets),
		caption: version?.tag
	}));
}
