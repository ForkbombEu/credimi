// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, test } from 'vitest';

import { Collections } from '@/pocketbase/types';
import { pocketbase as pb, pocketbaseCrud } from '@/v2';

describe('Pocketbase Record Form', () => {
	let form: pb.recordform.Instance<'credential_issuers'>;

	beforeEach(() => {
		form = new pb.recordform.Instance({
			collection: 'credential_issuers',
			mode: 'create',
			initialData: {},
			crud: new pocketbaseCrud.Instance('credential_issuers', {
				// TODO - Use array storage or kv storage for testing
				client: pocketbaseCrud.createMockClient()
			})
		});
	});

	test('should create a form', () => {
		expect(form).toBeDefined();
	});

	test('should change mode', async () => {
		await form.changeMode({
			mode: 'update',
			record: {
				id: '1',
				name: 'Test',
				created: '2021-01-01',
				description: 'Test',
				homepage_url: 'Test',
				imported: false,
				logo_url: 'Test',
				published: false,
				repo_url: 'Test',
				workflow_url: 'Test',
				collectionId: 'credential_issuers',
				owner: 'Test',
				updated: '2021-01-01',
				url: 'Test',
				collectionName: Collections.CredentialIssuers
			}
		});
		expect(form.currentMode).toEqual('update');
		console.log(form.form.values);
		expect(form.form.values).toMatchObject({
			name: 'Test'
		});
	});
});
