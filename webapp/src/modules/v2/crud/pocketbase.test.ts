// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, test, vi } from 'vitest';

import {
	Collections,
	type CredentialIssuersFormData,
	type CredentialIssuersResponse
} from '@/pocketbase/types';
import { task } from '@/v2';

import type { db, pocketbase as pb } from '..';

import { pocketbaseCrud } from '.';

//

function createMockClient<C extends db.CollectionName>(
	overrides: Partial<pb.TypedMockedRecordService<C>> = {}
): pb.TypedMockedClient {
	return {
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		collection: <C extends db.CollectionName>(collectionName: C) => {
			return {
				getOne: vi.fn(),
				getFullList: vi.fn(),
				create: vi.fn(),
				update: vi.fn(),
				delete: vi.fn(),
				...overrides
			} as pb.TypedMockedRecordService<C>;
		}
	};
}

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
		mockClient = createMockClient({
			create: vi.fn().mockResolvedValue(record),
			getOne: vi.fn().mockResolvedValue(record)
		});
		crud = new pocketbaseCrud.Instance('credential_issuers', { client: mockClient });
		const result = await task.run(crud.create(input));
		expect(result).toEqual(record);
	});

	// test('read handles PocketBase error', async () => {
	// 	mockCollection.getOne.mockRejectedValue(new Error('Not found'));

	// 	const taskPromise = task.run(crud.read('invalid'));
	// 	await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	// });

	// test('readAll returns all items', async () => {
	// 	const mockRecords = [
	// 		{ id: '1', name: 'User 1', username: 'user1' },
	// 		{ id: '2', name: 'User 2', username: 'user2' }
	// 	];
	// 	mockCollection.getFullList.mockResolvedValue(mockRecords);

	// 	const result = await task.run(crud.readAll());
	// 	expect(mockCollection.getFullList).toHaveBeenCalledWith({});
	// 	expect(result).toEqual(mockRecords);
	// });

	// test('create calls PocketBase create', async () => {
	// 	const mockInput = {
	// 		id: '2',
	// 		password: 'password',
	// 		tokenKey: 'token',
	// 		username: 'newuser',
	// 		name: 'New User'
	// 	};
	// 	const mockRecord = { id: '2', name: 'New User', username: 'newuser' };
	// 	mockCollection.create.mockResolvedValue(mockRecord);

	// 	const result = await task.run(crud.create(mockInput));
	// 	expect(mockCollection.create).toHaveBeenCalledWith(mockInput, {});
	// 	expect(result).toEqual(mockRecord);
	// });

	// test('update calls PocketBase update', async () => {
	// 	const mockInput = { name: 'Updated User' };
	// 	const mockRecord = { id: '1', name: 'Updated User', username: 'user1' };
	// 	mockCollection.update.mockResolvedValue(mockRecord);

	// 	const result = await task.run(crud.update('1', mockInput));
	// 	expect(mockCollection.update).toHaveBeenCalledWith('1', mockInput, {});
	// 	expect(result).toEqual(mockRecord);
	// });

	// test('delete calls PocketBase delete', async () => {
	// 	mockCollection.delete.mockResolvedValue(true);

	// 	const result = await task.run(crud.delete('1'));
	// 	expect(mockCollection.delete).toHaveBeenCalledWith('1', {});
	// 	expect(result).toBe(true);
	// });

	// test('uses default client when none provided', () => {
	// 	const crudWithDefaultClient = new pocketbaseCrud.Instance('users', {});
	// 	expect(crudWithDefaultClient.client).toBeDefined();
	// });

	// test('merges options correctly', async () => {
	// 	const crudWithOptions = new pocketbaseCrud.Instance('users', {
	// 		client: mockClient,
	// 		expand: 'profile'
	// 	});
	// 	mockCollection.getOne.mockResolvedValue({});

	// 	await task.run(crudWithOptions.read('1', { filter: 'active=true' }));
	// 	expect(mockCollection.getOne).toHaveBeenCalledWith('1', {
	// 		expand: 'profile',
	// 		filter: 'active=true'
	// 	});
	// });

	// test('handles create error', async () => {
	// 	mockCollection.create.mockRejectedValue(new Error('Validation failed'));

	// 	const taskPromise = task.run(
	// 		crud.create({
	// 			id: '3',
	// 			password: 'pass',
	// 			tokenKey: 'token',
	// 			username: 'user3'
	// 		})
	// 	);
	// 	await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	// });

	// test('handles update error', async () => {
	// 	mockCollection.update.mockRejectedValue(new Error('Not found'));

	// 	const taskPromise = task.run(crud.update('invalid', { name: 'Updated' }));
	// 	await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	// });

	// test('handles delete error', async () => {
	// 	mockCollection.delete.mockRejectedValue(new Error('Cannot delete'));

	// 	const taskPromise = task.run(crud.delete('invalid'));
	// 	await expect(taskPromise).rejects.toBeInstanceOf(t.BaseError);
	// });
});
