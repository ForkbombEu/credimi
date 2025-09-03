// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import keypairoomClient from '$zencode/keypairoom/keypairoomClient-8-9-10-11-12.zen?raw';
import keypairoomClientRecreateKeys from '$zencode/keypairoom/keypairoomClientRecreateKeys.zen?raw';
import matchKeys from '$zencode/keypairoom/match_pubkeys_secretkeys.zen?raw';
import _ from 'lodash';
import { zencode_exec } from 'zenroom';

import { pb } from '@/pocketbase';

import type { UserChallenges } from './userQuestions';
import type { PublicKeys } from './utils';

//

export interface Keyring {
	bitcoin: string;
	credential: string;
	ecdh: string;
	eddsa: string;
	es256: string;
	ethereum: string;
	reflow: string;
}

export interface Keypair {
	bitcoin_public_key: string;
	ecdh_public_key: string;
	eddsa_public_key: string;
	es256_public_key: string;
	ethereum_address: string;
	keyring: Keyring;
	reflow_public_key: string;
	seed: string;
}

//

type ZenroomProps = {
	data?: Record<string, unknown>;
	keys?: Record<string, unknown>;
	conf?: string;
};

export async function zencodeExec<T>(contract: string, props: ZenroomProps = {}): Promise<T> {
	const { result } = await zencode_exec(
		contract,
		_.mapValues(props, (v) => {
			if (typeof v == 'string') return v; // for conf
			return JSON.stringify(v);
		})
	);
	return JSON.parse(result);
}

//

export async function generateKeypair(
	email: string,
	HMAC: string,
	userChallenges: UserChallenges
): Promise<Keypair> {
	return await zencodeExec<Keypair>(keypairoomClient, {
		data: {
			userChallenges,
			username: email,
			'seedServerSideShard.HMAC': HMAC
		}
	});
}

export async function regenerateKeypair(seed: string, HMAC: string): Promise<Keypair> {
	return await zencodeExec<Keypair>(keypairoomClientRecreateKeys, {
		data: {
			seed,
			'seedServerSideShard.HMAC': HMAC
		}
	});
}

//

export async function matchPublicAndPrivateKeys(publicKeys: PublicKeys, privateKeys: Keyring) {
	const renamedPublicKeys = _.mapKeys(publicKeys, (v, k) => `${k}_backend`); // As required by contract
	await zencodeExec(matchKeys, {
		data: {
			keyring: privateKeys
		},
		keys: renamedPublicKeys
	});
}

//

export async function getHMAC(email: string): Promise<string> {
	const response = await pb.send('/api/keypairoom-server', {
		body: JSON.stringify({ userData: { email } }),
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		}
	});
	return response.hmac;
}

//

export const KEYRING_STORAGE_KEY = 'keyring';

export function saveKeyringToLocalStorage(keyring: Keyring) {
	localStorage.setItem(KEYRING_STORAGE_KEY, JSON.stringify(keyring));
}

export function getKeyringFromLocalStorage(): Keyring | null {
	if (!browser) throw new Error('getKeyringFromLocalStorage() must be called from the browser');
	const keyring = localStorage.getItem(KEYRING_STORAGE_KEY);
	if (keyring) {
		return JSON.parse(keyring);
	}
	return null;
}
