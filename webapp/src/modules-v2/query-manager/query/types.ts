// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { NonEmptyTuple, Simplify } from 'type-fest';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { CollectionExpands, CollectionResponses } from '@/pocketbase/types';
import type { KeyOf } from '@/utils/types';

import type { QueryParams } from './params';

/* Main */

export type Query<C extends CollectionName, E extends ExpandOption<C> = never> = BaseQuery<C, E> &
	QueryParams<C> &
	Options;

export type BaseQuery<C extends CollectionName, E extends ExpandOption<C> = never> = {
	collection: C;
	expand?: E;
};

export type Options = Partial<{
	fetch: typeof fetch;
	requestKey: string | null;
	url: URL;
}>;

// Expand

export type ExpandOption<C extends CollectionName> =
	Expand<C> extends Record<string, never> ? never : NonEmptyTuple<ExpandKey<C>>;

type Expand<C extends CollectionName> = CollectionExpands[C];
type ExpandKey<C extends CollectionName> = KeyOf<Expand<C>>;

type ResolvedExpand<C extends CollectionName, E extends ExpandOption<C> = never> = E extends never
	? never
	: Simplify<Pick<Expand<C>, E[number]>>;

// Response

export type Response<C extends CollectionName, E extends ExpandOption<C> = never> = {
	records: ResponseRecord<C, E>[];
	totalItems: number;
	rootParams: QueryParams<C>;
	urlParams: QueryParams<C>;
	requestKey?: string | null;
};

export type ResponseRecord<C extends CollectionName, E extends ExpandOption<C> = never> =
	ResolvedExpand<C, E> extends never
		? CollectionResponses[C]
		: Simplify<CollectionResponses[C] & { expand: ResolvedExpand<C, E> }>;
