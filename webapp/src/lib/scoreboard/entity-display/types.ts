// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';

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

export type AvatarData = {
	src?: string;
	fallback: string;
	alt: string;
};

export type ChildLink = {
	label: string;
	href: string;
	avatar?: AvatarData;
};

export type Item = {
	key: string;
	name: string;
	href: string;
	avatar?: AvatarData;
	kind?: EntityData;
	caption?: string;
	children?: ChildLink[];
};

export type Layout = 'avatar-only' | 'links-only' | 'compact' | 'full';

export type Align = 'start' | 'end';

export type PocketbaseEntity =
	| WalletsResponse
	| CredentialIssuersResponse
	| VerifiersResponse
	| UseCasesVerificationsResponse
	| CredentialsResponse
	| CustomChecksResponse
	| PipelinesResponse;
