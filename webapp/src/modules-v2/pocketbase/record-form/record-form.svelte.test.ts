// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { form as f, pocketbase as pb, pocketbaseCrud } from '#';
import { beforeEach, describe, expect, test } from 'vitest';

import { Collections, type CredentialIssuersResponse } from '@/pocketbase/types';

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
		f.mountTestComponent(form.form);
	});

	test('should create a form', () => {
		expect(form).toBeDefined();
	});

	// TODO - Does not work because superform must be mounted before updates can work
	test('should change mode', async () => {
		const record: CredentialIssuersResponse = {
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
		};
		await form.changeMode({
			mode: 'update',
			record
		});
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		const { collectionId, collectionName, ...rest } = record;
		expect(form.currentMode).toEqual('update');
		expect(form.form.values).toEqual(rest);
	});
});
