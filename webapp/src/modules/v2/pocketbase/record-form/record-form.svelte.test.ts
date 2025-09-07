// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, test } from 'vitest';

import { pocketbase as pb, pocketbaseCrud } from '@/v2';

describe('Pocketbase Record Form', () => {
	let form: pb.recordform.Instance<'credential_issuers'>;

	// beforeEach(() => {

	// });

	test('should create a form', () => {
		form = new pb.recordform.Instance({
			collection: 'credential_issuers',
			mode: 'create',
			initialData: {},
			crud: new pocketbaseCrud.Instance('credential_issuers', {
				// TODO - Use array storage or kv storage for testing
				client: pocketbaseCrud.createMockClient()
			})
		});
		expect(form).toBeDefined();
	});
});
