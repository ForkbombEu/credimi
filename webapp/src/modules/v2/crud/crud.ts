// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as task from 'true-myth/task';

import type { CollectionName } from '@/pocketbase/collections-models';
import type * as db from '@/pocketbase/types';

import { pocketbase as pb, type types as t } from '../.';
import { BaseError } from '../types';

/* Crud */

export type DefaultConfig = {
	type: object;
	input?: object;
	key: string | number;
	options?: object;
};

export type Task<Config extends DefaultConfig = DefaultConfig> = task.Task<
	Config['type'] & { key: Config['key'] },
	BaseError
>;

export interface Crud<c extends DefaultConfig> {
	read(key: c['key'], options?: c['options']): Task<c>;
	readAll(options?: c['options']): task.Task<c['type'][], t.BaseError>;
	create(input: c['input'], options?: c['options']): Task<c>;
	update(key: c['key'], input: Partial<c['input']>, options?: c['options']): Task<c>;
	delete(key: c['key'], options?: c['options']): task.Task<boolean, t.BaseError>;
}

/* ArrayCrud */

type ArrayCrudConfig<T extends object = object, Input extends object = T> = {
	type: T;
	input: Input;
	key: number;
};

export class ArrayCrud<Config extends ArrayCrudConfig> implements Crud<Config> {
	constructor(private readonly items: Config['type'][] = []) {}

	read(key: number): Task<Config> {
		const item = this.items.at(key);
		if (!item) return task.reject(new Error('Item not found'));
		return task.resolve({ key, ...item });
	}

	readAll(): task.Task<Config['type'][], BaseError> {
		return task.resolve(this.items);
	}

	create(input: Config['input']): Task<Config> {
		const item = { ...input, key: this.items.length };
		this.items.push(item);
		return task.resolve({ key: item.key, value: item });
	}

	update(key: Config['key'], input: Partial<Config['input']>): Task<Config> {
		const item = this.items.at(key);
		if (!item) return task.reject(new Error('Item not found'));
		return task.resolve({ key, ...item, ...input });
	}

	delete(key: Config['key']): task.Task<boolean, Error> {
		// @ts-expect-error - we want to remove the item
		this.items[key] = undefined;
		return task.resolve(true);
	}
}

/* Pocketbase Crud */

type RecordPbCrudOptions = Partial<pb.QueryOptions> & {
	client?: pb.Pocketbase;
};

export type RecordPbCrudConfig<C extends CollectionName> = {
	type: db.CollectionResponses[C];
	input: db.CollectionFormData[C];
	key: string;
	options?: RecordPbCrudOptions;
};

export class RecordPbCrud<
	C extends CollectionName,
	Config extends RecordPbCrudConfig<C> = RecordPbCrudConfig<C>
> implements Crud<Config>
{
	constructor(
		public readonly collection: C,
		public readonly options: Config['options']
	) {}

	get client() {
		return this.options?.client ?? pb.defaultClient;
	}

	private getClientOptions(overrides: Config['options'] = {}) {
		if (!this.options) return overrides;
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		const { client, ...rest } = this.options;
		return { ...rest, ...overrides };
	}

	read(key: Config['key'], options: Config['options'] = {}): Task<Config> {
		return task.fromPromise(
			this.client.collection(this.collection).getOne(key, this.getClientOptions(options)),
			(e) => new BaseError(e)
		);
	}

	readAll(options: Config['options'] = {}): task.Task<Config['type'][], BaseError> {
		return task.fromPromise(
			this.client.collection(this.collection).getFullList(this.getClientOptions(options)),
			(e) => new BaseError(e)
		);
	}

	create(input: Config['input'], options: Config['options'] = {}): Task<Config> {
		return task.fromPromise(
			this.client.collection(this.collection).create(input, this.getClientOptions(options)),
			(e) => new BaseError(e)
		);
	}

	update(
		key: Config['key'],
		input: Partial<Config['input']>,
		options: Config['options'] = {}
	): Task<Config> {
		return task.fromPromise(
			this.client
				.collection(this.collection)
				.update(key, input, this.getClientOptions(options)),
			(e) => new BaseError(e)
		);
	}

	delete(key: Config['key'], options: Config['options'] = {}): task.Task<boolean, BaseError> {
		return task.fromPromise(
			this.client.collection(this.collection).delete(key, this.getClientOptions(options)),
			(e) => new BaseError(e)
		);
	}
}

/* Record Crud */

// export class RecordCrud<C extends CollectionName, Options extends RecordCrudOptions<C>>
// 	implements Crud<Options> {
// 		read(key: Options['key']): Task<Options> {
// 			return task.resolve({ key, ...this.items.at(key) });
// 		}
// 	}
