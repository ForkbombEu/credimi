// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export async function fetchCredentialIssuer(url: string) {
	await pb.send('/credentials_issuers/start-check', {
		method: 'POST',
		body: {
			credentialIssuerUrl: url
		}
	});
}

export async function getCredentialIssuerByUrl(url: string, organizationId: string) {
	const credentialIssuers = await pb
		.collection('credential_issuers')
		.getFullList({ filter: `url = '${url}' && owner = '${organizationId}'` });

	if (credentialIssuers.length != 1) {
		throw new Error('Credential issuer not found');
	}

	return credentialIssuers[0];
}
