// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { types as t, task } from '#';
import { beforeEach, describe, expect, test } from 'vitest';

import { arrayCrud } from '.';

//

describe('ArrayCrud', () => {
	let crud: arrayCrud.Instance<{
		type: { name: string };
	}>;

	beforeEach(() => {
		crud = new arrayCrud.Instance([{ name: 'Item 0' }, { name: 'Item 1' }]);
	});

	test('read existing item', async () => {
		const item = await task.run(crud.read(0));
		expect(item).toEqual({ index: 0, name: 'Item 0' });
	});

	test('read non-existing item throws error', async () => {
		const taskPromise = task.run(crud.read(10));
		await expect(taskPromise).rejects.toBeInstanceOf(t.NotFoundError);
	});

	test('readAll returns all items', async () => {
		const result = await task.run(crud.readAll());
		expect(result).toEqual([
			{ index: 0, name: 'Item 0' },
			{ index: 1, name: 'Item 1' }
		]);
	});

	test('create adds new item', async () => {
		const result = await task.run(crud.create({ name: 'New Item' }));
		expect(result).toEqual({ index: 2, name: 'New Item' });
	});

	test('update existing item', async () => {
		const result = await task.run(crud.update(0, { name: 'Updated Item' }));
		expect(result).toEqual({ index: 0, name: 'Updated Item' });
	});

	test('update non-existing item throws error', async () => {
		const taskPromise = task.run(crud.update(10, { name: 'Updated' }));
		await expect(taskPromise).rejects.toBeInstanceOf(t.NotFoundError);
	});

	test('delete marks item as undefined', async () => {
		const result = await task.run(crud.delete(0));
		expect(result).toBe(true);
		await expect(task.run(crud.read(0))).rejects.toBeInstanceOf(t.NotFoundError);
	});

	test('delete non-existing item throws error', async () => {
		const taskPromise = task.run(crud.delete(10));
		await expect(taskPromise).rejects.toBeInstanceOf(t.NotFoundError);
	});
});
