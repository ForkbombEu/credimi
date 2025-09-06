// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, test } from 'vitest';

import { CredentialIssuerManager } from './credential-issuer-manager.svelte.js';

//

describe.skip('CredentialIssuerManager', () => {
	let manager: CredentialIssuerManager;

	beforeEach(() => {
		manager = new CredentialIssuerManager();
	});

	test('Effect', () => {
		const cleanup = $effect.root(() => {
			console.log(manager);
		});

		cleanup();
	});
});
