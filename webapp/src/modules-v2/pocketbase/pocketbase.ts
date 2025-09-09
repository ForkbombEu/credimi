// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { db } from '#';
import type * as pb from 'pocketbase';
import type { Simplify } from 'type-fest';
import type * as vi from 'vitest';

import GenericPocketBase from 'pocketbase';

import type { KeyOf } from '@/utils/types';

import { pb as defaultPocketbaseClient } from '@/pocketbase';

/* Types */

export type Pocketbase = GenericPocketBase;
export const defaultClient = defaultPocketbaseClient;

export type QueryOptions = Simplify<
	pb.FullListOptions | pb.RecordListOptions | pb.RecordOptions | pb.ListOptions
>;
export type RecordOptions = pb.RecordOptions;
export type ID = string;

/* Record */

export type BaseRecord<C extends db.CollectionName> = Omit<
	db.CollectionRecords[C],
	'id' | 'created' | 'updated'
>;

export type Field<C extends db.CollectionName> = KeyOf<db.CollectionRecords[C]>;

/* Client & Record Service */

export interface CoreRecordService<T extends object, Input extends object> {
	getOne: (id: string, options?: pb.RecordOptions) => Promise<T>;
	getFullList: (options?: pb.FullListOptions) => Promise<T[]>;
	create: (data: Input, options?: pb.RecordOptions) => Promise<T>;
	update: (id: string, data: Partial<Input>, options?: pb.RecordOptions) => Promise<T>;
	delete: (id: string, options?: pb.CommonOptions) => Promise<boolean>;
}

export type CoreClient = {
	collection: <T extends object, Input extends object>(
		collectionName: string
	) => CoreRecordService<T, Input>;
};

export type TypedCoreClient = {
	collection: <C extends db.CollectionName>(
		collectionName: C
	) => CoreRecordService<db.CollectionResponses[C], db.CollectionFormData[C]>;
};

export type MockedRecordService<T extends object, Input extends object> = {
	[K in keyof CoreRecordService<T, Input>]: vi.Mock<CoreRecordService<T, Input>[K]>;
};

export type TypedMockedRecordService<C extends db.CollectionName> = MockedRecordService<
	db.CollectionResponses[C],
	db.CollectionFormData[C]
>;

export type TypedMockedClient = {
	collection: <C extends db.CollectionName>(collectionName: C) => TypedMockedRecordService<C>;
};
