// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { db } from '#';
import type { Simplify } from 'type-fest';

import { crud, pocketbase as pb, task } from '#';

/* Pocketbase Crud */

export type ResolvedConfig<C extends db.CollectionName = db.CollectionName> = {
	type: db.CollectionResponses[C];
	input: Simplify<Omit<db.CollectionFormData[C], 'id' | 'created' | 'updated'>>;
	key: string;
	keyName: 'id';
	options?: pb.QueryOptions;
};

type Task<T> = task.WithError<T>;

export class Instance<
	C extends db.CollectionName,
	RConfig extends ResolvedConfig<C> = ResolvedConfig<C>,
	FormData extends db.CollectionFormData[C] = db.CollectionFormData[C],
	Response extends db.CollectionResponses[C] = db.CollectionResponses[C]
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

	private getClientOptions(overrides: pb.QueryOptions = {}) {
		if (!this.options) return overrides;
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		const { client, ...rest } = this.options;
		return { ...rest, ...overrides };
	}

	read(id: pb.ID, options: pb.RecordOptions = {}): Task<Response> {
		return task.withError(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).getOne(id, this.getClientOptions(options))
		);
	}

	readAll(options: pb.QueryOptions = {}): Task<Response[]> {
		return task.withError(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).getFullList(this.getClientOptions(options))
		);
	}

	create(input: FormData, options: pb.RecordOptions = {}): Task<Response> {
		return task.withError(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).create(input, this.getClientOptions(options))
		);
	}

	update(id: pb.ID, input: Partial<FormData>, options: pb.RecordOptions = {}): Task<Response> {
		return task.withError(
			this.client
				// @ts-expect-error - Slight type mismatch
				.collection(this.collection)
				.update(id, input, this.getClientOptions(options))
		);
	}

	delete(id: pb.ID, options: pb.RecordOptions = {}): Task<boolean> {
		return task.withError(
			// @ts-expect-error - Slight type mismatch
			this.client.collection(this.collection).delete(id, this.getClientOptions(options))
		);
	}
}
