// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';

import type {
	CredentialIssuersResponse,
	CredentialsResponse,
	UseCasesVerificationsResponse,
	VerifiersResponse,
	WalletsResponse
} from '@/pocketbase/types';

//

export type Entity =
	| WalletsResponse
	| CredentialIssuersResponse
	| VerifiersResponse
	| UseCasesVerificationsResponse
	| CredentialsResponse;

export function getEntityHref(entity: Entity): string {
	return `/marketplace/${entity.collectionName}/${getPath(entity)}`;
}
