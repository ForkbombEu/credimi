// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export function createIntentUrl(issuer: string | undefined, type: string): string {
	const data = {
		credential_configuration_ids: [type],
		credential_issuer: issuer
	};
	const credentialOffer = encodeURIComponent(JSON.stringify(data));
	return `openid-credential-offer://?credential_offer=${credentialOffer}`;
}
