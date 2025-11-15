// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { OpenidCredentialIssuer } from './openid-credential-issuer.generated';

export type { OpenidCredentialIssuer };
export type CredentialConfiguration =
	OpenidCredentialIssuer['credential_configurations_supported'][string];
export type CredentialDefinition = NonNullable<CredentialConfiguration['credential_definition']>;
export type CredentialSubject = NonNullable<CredentialDefinition['credentialSubject']>;
