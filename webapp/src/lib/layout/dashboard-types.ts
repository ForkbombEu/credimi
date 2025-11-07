// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type {
	CredentialIssuersResponse,
	CredentialsResponse,
	CustomChecksResponse,
	PipelinesResponse,
	UseCasesVerificationsResponse,
	VerifiersResponse,
	WalletsResponse
} from '@/pocketbase/types';

export type DashboardRecord =
	| PipelinesResponse
	| CredentialIssuersResponse
	| CredentialsResponse
	| VerifiersResponse
	| CustomChecksResponse
	| UseCasesVerificationsResponse
	| WalletsResponse;
