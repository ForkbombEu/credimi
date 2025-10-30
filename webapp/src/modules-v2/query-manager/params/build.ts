// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RecordListOptions } from 'pocketbase';

import { Record } from 'effect';

import type { CollectionName } from '@/pocketbase/collections-models';

import { ensureArray } from '@/utils/other';

import type { ExcludeParam, FilterParam, QueryParams, SearchParam, SortParam } from './types';

import { ensureSortArray } from './utils';

//

export function build<C extends CollectionName = never>(query: QueryParams<C>): RecordListOptions {
	const { sort, search, filter, excludeIDs, page = 1, perPage = 25 } = query;
	const listOptions: RecordListOptions = {
		page,
		perPage
	};

	if (sort) listOptions.sort = buildSortParam(sort);

	const filters: string[] = [];
	if (filter) filters.push(buildFilterParam(filter));
	if (search) filters.push(buildSearchParam(search));
	if (excludeIDs) filters.push(buildExcludeParam(excludeIDs));
	if (filters.length > 0) listOptions.filter = filters.map((i) => `(${i})`).join(AND);

	return Record.filter(listOptions as never, Boolean);
}

// Partials

function buildSortParam(sort: SortParam<never>): string {
	const base = ensureSortArray(sort);
	if (base.length === 0) throw new EmptyParamError('sort');
	return base
		.map(([field, order]) => {
			const prefix = order == 'ASC' ? '+' : '-';
			return `${prefix}${field}`;
		})
		.join(',');
}

function buildFilterParam(filter: FilterParam): string {
	const base = ensureArray(filter);
	if (base.length === 0) throw new EmptyParamError('filter');
	const items = base.map((f) => {
		if (typeof f === 'string') return f;
		return f.expressions.join(f.mode == 'AND' ? AND : OR);
	});
	if (items.length == 1) return items[0];
	return items.map((i) => `(${i})`).join(AND);
}

function buildSearchParam(search: SearchParam<never>): string {
	const [text, fields] = search;
	if (fields.length === 0) throw new EmptyParamError('search');
	const fieldsArray = ensureArray(fields);
	return fieldsArray.map((f) => `${f} ~ ${QUOTE}${text}${QUOTE}`).join(OR);
}

function buildExcludeParam(exclude: ExcludeParam): string {
	const base = ensureArray(exclude);
	if (base.length === 0) throw new EmptyParamError('exclude');
	return base.map((id) => `id != ${QUOTE}${id}${QUOTE}`).join(AND);
}

// Utils

const QUOTE = '"';
const OR = ' || ';
const AND = ' && ';

class EmptyParamError extends Error {
	constructor(message?: string) {
		super('Empty param: ' + message);
	}
}
