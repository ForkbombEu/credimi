// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import {
	Collections,
	type CredentialIssuersRecord,
	type CredentialsRecord,
	type UseCasesVerificationsRecord,
	type UsersResponse,
	type VerifiersRecord,
	type WalletActionsRecord,
	type WalletsRecord
} from '@/pocketbase/types';

export function getUserDisplayName(user: UsersResponse) {
	return user.name ? user.name : user.username ? user.username : user.email;
}

// Configuration mapping collections to their parent field names using type-safe field extraction
const COLLECTION_PARENT_FIELD_MAP = {
	[Collections.CredentialIssuers]: 'owner' as keyof CredentialIssuersRecord,
	[Collections.Credentials]: 'credential_issuer' as keyof CredentialsRecord,
	[Collections.Verifiers]: 'owner' as keyof VerifiersRecord,
	[Collections.UseCasesVerifications]: 'verifier' as keyof UseCasesVerificationsRecord,
	[Collections.Wallets]: 'owner' as keyof WalletsRecord,
	[Collections.WalletActions]: 'wallet' as keyof WalletActionsRecord
} as const;

export async function checkNameUniqueness(
	collectionName: keyof typeof COLLECTION_PARENT_FIELD_MAP,
	name: string,
	parentId: string,
	excludeId?: string
): Promise<boolean> {
	const parentField = COLLECTION_PARENT_FIELD_MAP[collectionName];

	try {
		const filter = excludeId
			? `${parentField} = "${parentId}" && name = "${name}" && id != "${excludeId}"`
			: `${parentField} = "${parentId}" && name = "${name}"`;

		const records = await pb.collection(collectionName).getList(1, 1, { filter });
		return records.totalItems === 0;
	} catch (error) {
		// Check if this is an auto-cancellation error
		if (error && typeof error === 'object' && 'isAbort' in error && error.isAbort) {
			console.log('Name uniqueness check was cancelled (auto-cancellation)');
			return true; // Assume unique when cancelled to avoid blocking the user
		}

		console.error('Error checking name uniqueness:', error);
		return false; // Assume not unique on other errors for safety
	}
}
