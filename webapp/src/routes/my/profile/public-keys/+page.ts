// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { loadFeatureFlags } from '@/features';
import { getUserPublicKeys, type PublicKeys } from '@/keypairoom/utils';

export const load = async ({ fetch }) => {
	const { KEYPAIROOM } = await loadFeatureFlags(fetch);
	if (!KEYPAIROOM) error(404);

	const publicKeysResponse = await getUserPublicKeys();
	if (!publicKeysResponse) {
		return undefined;
	}

	const publicKeys: PublicKeys = {
		ecdh_public_key: publicKeysResponse.ecdh_public_key,
		eddsa_public_key: publicKeysResponse.eddsa_public_key,
		reflow_public_key: publicKeysResponse.reflow_public_key,
		es256_public_key: publicKeysResponse.es256_public_key,
		bitcoin_public_key: publicKeysResponse.bitcoin_public_key,
		ethereum_address: publicKeysResponse.ethereum_address
	};

	return {
		publicKeys
	};
};
