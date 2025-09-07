// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, test } from 'vitest';

import { pocketbase as pb } from '@/v2';

describe('Pocketbase Record Form', () => {
	let form: pb.record.Form<'credential_issuers'>;

	// beforeEach(() => {

	// });

	test('should create a form', () => {
		form = new pb.record.Form({
			collection: 'credential_issuers',
			mode: 'create',
			initialData: {}
		});
		expect(form).toBeDefined();
	});
});
