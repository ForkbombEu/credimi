// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { pocketbase as pb } from '#/ideas';

import { describe, expect, test } from 'vitest';

import { MockFile, recordToFormData } from './functions';

//

describe('mockFile', () => {
	test('should mock a file', () => {
		const filename = 'test.txt';
		const mimeType = 'text/plain';
		const file = new MockFile(filename, { mimeType });
		expect(file).toBeDefined();
		expect(file.name).toBe(filename);
		expect(file.type).toBe(mimeType);
		expect(file.size).toBe(0);
	});
});

describe('recordToFormData', () => {
	test('should convert a simple record to form data', () => {
		const record: pb.BaseRecord<'credentials'> = {
			name: 'Test',
			owner: 'test'
		};
		const formData = recordToFormData('credentials', record);
		expect(formData).toEqual({
			name: 'Test',
			owner: 'test'
		});
	});

	test('should convert a record with a file to form data', () => {
		const wallet: pb.BaseRecord<'wallets'> = {
			name: 'Test',
			owner: 'test',
			logo: 'test.png'
		};
		const formData = recordToFormData('wallets', wallet);
		expect(formData.logo).toBeInstanceOf(MockFile);
	});

	test('should convert a record with a json field to form data', () => {
		const json = {
			test: 'test'
		};
		const record: pb.BaseRecord<'custom_checks'> = {
			name: 'Test',
			owner: 'test',
			standard_and_version: 'test/1.0.0',
			yaml: 'test.yaml',
			input_json_schema: json
		};
		const formData = recordToFormData('custom_checks', record);
		expect(formData.input_json_schema).toEqual(JSON.stringify(json));
	});

	// TODO - Add more tests
});
