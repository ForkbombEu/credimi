// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export async function fetchCredentialIssuer(url: string) {
	await pb.send('/api/credentials_issuers/start-check', {
		method: 'POST',
		body: {
			credentialIssuerUrl: url
		}
	});
}
