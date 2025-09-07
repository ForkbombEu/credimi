// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pocketbase as pb, pocketbaseCrud, types as t, task } from '#';
import { beforeEach, describe, expect, test, vi } from 'vitest';

import {
	Collections,
	type CredentialIssuersFormData,
	type CredentialIssuersResponse
} from '@/pocketbase/types';

import { createMockClient } from './pocketbase';

//

describe('PocketBaseCrud', () => {
	let mockClient: pb.TypedMockedClient;
	let crud: pocketbaseCrud.Instance<'credential_issuers'>;
	let input: CredentialIssuersFormData;
	let record: CredentialIssuersResponse;

	beforeEach(() => {
		mockClient = createMockClient();
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });
		input = {
			owner: 'test',
			url: 'test'
		};
		record = {
			...input,
			id: '1',
			name: 'test',
			created: '2021-01-01',
			updated: '2021-01-01',
			description: 'test',
			homepage_url: 'test',
			imported: false,
			logo_url: 'test',
			published: false,
			repo_url: 'test',
			workflow_url: 'test',
			collectionId: 'credential_issuers',
			collectionName: Collections.CredentialIssuers
		};
	});

	test('read existing item', async () => {
		// TODO - Use array or kv storage to test better
		mockClient = createMockClient({
			getOne: vi.fn().mockResolvedValue(record)
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });
		const result = await task.run(crud.read('1'));
		expect(result).toEqual(record);
	});

	test('read handles PocketBase error', async () => {
		mockClient = createMockClient({
			getOne: vi.fn().mockRejectedValue(new Error('Not found'))
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const taskPromise = task.run(crud.read('invalid'));
		await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	});

	test('readAll returns all items', async () => {
		const mockRecords = [record, { ...record, id: '2', name: 'test2' }];
		mockClient = createMockClient({
			getFullList: vi.fn().mockResolvedValue(mockRecords)
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const result = await task.run(crud.readAll());
		expect(result).toEqual(mockRecords);
	});

	test('create calls PocketBase create', async () => {
		mockClient = createMockClient({
			create: vi.fn().mockResolvedValue(record)
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const result = await task.run(crud.create(input));
		expect(result).toEqual(record);
	});

	test('update calls PocketBase update', async () => {
		const updateInput = { name: 'Updated Name' };
		const updatedRecord = { ...record, name: 'Updated Name' };
		mockClient = createMockClient({
			update: vi.fn().mockResolvedValue(updatedRecord)
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const result = await task.run(crud.update('1', updateInput));
		expect(result).toEqual(updatedRecord);
	});

	test('delete calls PocketBase delete', async () => {
		mockClient = createMockClient({
			delete: vi.fn().mockResolvedValue(true)
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const result = await task.run(crud.delete('1'));
		expect(result).toBe(true);
	});

	test('uses default client when none provided', () => {
		const crudWithDefaultClient = new pocketbaseCrud.Instance('credential_issuers', {});
		expect(crudWithDefaultClient.client).toBeDefined();
	});

	test('merges options correctly', async () => {
		mockClient = createMockClient({
			getOne: vi.fn().mockResolvedValue(record)
		});
		const crudWithOptions = new pocketbaseCrud.Instance('credential_issuers', {
			client: mockClient,
			expand: 'owner'
		});

		await task.run(crudWithOptions.read('1', { filter: 'published=true' }));
		const mockCollection = mockClient.collection('credential_issuers');
		expect(mockCollection.getOne).toHaveBeenCalledWith('1', {
			expand: 'owner',
			filter: 'published=true'
		});
	});

	test('handles create error', async () => {
		mockClient = createMockClient({
			create: vi.fn().mockRejectedValue(new Error('Validation failed'))
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const taskPromise = task.run(crud.create(input));
		await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	});

	test('handles update error', async () => {
		mockClient = createMockClient({
			update: vi.fn().mockRejectedValue(new Error('Not found'))
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const taskPromise = task.run(crud.update('invalid', { name: 'Updated' }));
		await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	});

	test('handles delete error', async () => {
		mockClient = createMockClient({
			delete: vi.fn().mockRejectedValue(new Error('Cannot delete'))
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });

		const taskPromise = task.run(crud.delete('invalid'));
		await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	});
});
