// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';

import type {
	CredentialIssuersResponse,
	CredentialsResponse,
	CustomChecksResponse,
	PipelinesResponse,
	UseCasesVerificationsResponse,
	VerifiersResponse,
	WalletsResponse
} from '@/pocketbase/types';

//

export type RelatedEntity =
	| WalletsResponse
	| CredentialIssuersResponse
	| VerifiersResponse
	| UseCasesVerificationsResponse
	| CredentialsResponse
	| CustomChecksResponse
	| PipelinesResponse;

export function getRelatedEntityHref(entity: RelatedEntity): string {
	return `/marketplace/${entity.collectionName}/${getPath(entity)}`;
}
