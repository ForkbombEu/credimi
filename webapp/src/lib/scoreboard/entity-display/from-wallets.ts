// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities } from '$lib/global';

import type { Item } from './types';
import type { PocketbaseEntity } from './types';

import { fromPocketbaseEntity } from './from-pocketbase';

//

export type WalletRow = {
	wallet: PocketbaseEntity;
	version?: { wallet?: string; tag?: string };
};

export function fromWalletRows(rows: WalletRow[]): Item[] {
	return rows.map(({ wallet, version }) => ({
		...fromPocketbaseEntity(wallet, entities.wallets),
		caption: version?.tag
	}));
}
