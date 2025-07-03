// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { type Data, type UsersPublicKeysRecord } from '@/pocketbase/types';
import _ from 'lodash';
import type { Keypair } from './keypair';
import { createSessionStorageHandlers } from '@/utils/storage';
import { String } from 'effect';

export type PublicKeys = Omit<Data<UsersPublicKeysRecord>, 'owner'>;

export function getPublicKeysFromKeypair(keypair: Keypair): Data<PublicKeys> {
	const publicKeys = _.cloneDeep(keypair);
	// @ts-expect-error Cannot use delete on required field
	delete publicKeys.seed;
	// @ts-expect-error Cannot use delete on required field
	delete publicKeys.keyring;
	return publicKeys;
}

export async function getUserPublicKeys(userId: string | undefined = undefined, fetchFn = fetch) {
	const id = userId ?? pb.authStore.record?.id ?? '';
	if (String.isEmpty(id)) throw new Error('Missing user ID');
	try {
		return await pb
			.collection('users_public_keys')
			.getFirstListItem(`owner.id = '${id}'`, { fetch: fetchFn });
	} catch {
		return undefined;
	}
}

export async function saveUserPublicKeys(userId: string, publicKeys: PublicKeys) {
	const data: Data<UsersPublicKeysRecord> = {
		...publicKeys,
		owner: userId
	};
	await pb.collection('users_public_keys').create(data);
}

//

export const RegenerateKeyringSession = createSessionStorageHandlers('keypairoom_regenerate');
