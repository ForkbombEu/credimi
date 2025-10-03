// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CredentialsRecord } from '@/pocketbase/types';

//

export function createIntentUrl(
	credential: CredentialsRecord,
	credentialIssuerUrl: string
): string {
	if (!credential.name) throw new Error('Credential name is required');
	const data = {
		credential_configuration_ids: [credential.name],
		credential_issuer: credentialIssuerUrl
	};

	const credentialOffer = encodeURIComponent(JSON.stringify(data));
	return `openid-credential-offer://?credential_offer=${credentialOffer}`;
}
