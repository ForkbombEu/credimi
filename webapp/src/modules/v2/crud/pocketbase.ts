// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Simplify } from 'type-fest';

import * as db from '@/pocketbase/types';
import { crud } from '@/v2';

import { pocketbase as pb, types as t, task } from '..';

/* Pocketbase Crud */

export type ResolvedConfig<C extends db.CollectionName = db.CollectionName> = {
	type: db.CollectionResponses[C];
	input: Simplify<Omit<db.CollectionFormData[C], 'id' | 'created' | 'updated'>>;
	key: string;
	keyName: 'id';
	options?: pb.QueryOptions;
};

export class Instance<
	C extends db.CollectionName,
	RConfig extends ResolvedConfig<C> = ResolvedConfig<C>
> implements crud.Crud<RConfig>
{
	constructor(
		public readonly collection: C,
		public readonly options: RConfig['options'] & {
			client?: pb.TypedCoreClient;
		} = {}
	) {}

	get client() {
		return this.options?.client ?? pb.defaultClient;
	}

	private getClientOptions(overrides: RConfig['options'] = {}) {
		if (!this.options) return overrides;
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		const { client, ...rest } = this.options;
		return { ...rest, ...overrides };
	}

	read(key: RConfig['key'], options: RConfig['options'] = {}): crud.Task<RConfig> {
		return task.fromPromise(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).getOne(key, this.getClientOptions(options)),
			(e) => new t.BaseError(e)
		);
	}

	readAll(options: RConfig['options'] = {}): task.Task<crud.Record<RConfig>[], t.BaseError> {
		return task.fromPromise(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).getFullList(this.getClientOptions(options)),
			(e) => new t.BaseError(e)
		);
	}

	create(input: RConfig['input'], options: RConfig['options'] = {}): crud.Task<RConfig> {
		return task.fromPromise(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).create(input, this.getClientOptions(options)),
			(e) => new t.BaseError(e)
		);
	}

	update(
		key: RConfig['key'],
		input: Partial<RConfig['input']>,
		options: RConfig['options'] = {}
	): crud.Task<RConfig> {
		return task.fromPromise(
			this.client
				// @ts-expect-error - Slight type mismatch
				.collection(this.collection)
				.update(key, input, this.getClientOptions(options)),
			(e) => new t.BaseError(e)
		);
	}

	delete(key: RConfig['key'], options: RConfig['options'] = {}): task.Task<boolean, t.BaseError> {
		return task.fromPromise(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).delete(key, this.getClientOptions(options)),
			(e) => new t.BaseError(e)
		);
	}
}
