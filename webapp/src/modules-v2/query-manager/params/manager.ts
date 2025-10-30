// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type CollectionName } from '@/pocketbase/collections-models';

import type { ExcludeParam, QueryParams } from './types';

import { DEFAULT_PER_PAGE } from './utils';

/* Editor */

export class Manager<C extends CollectionName> {
	constructor(private params: QueryParams<C>) {}

	//

	setPageSize(perPage: number) {
		this.params.perPage = perPage;
	}

	getPageSize() {
		return this.params.perPage ?? DEFAULT_PER_PAGE;
	}

	//

	setSearch(text: string) {
		if (!this.params.search) return;
		this.params.search[0] = text;
	}

	clearSearch() {
		this.params.search = undefined;
	}

	hasSearch() {
		return Boolean(this.params.search);
	}

	//

	addExclude(exclude: ExcludeParam) {
		if (!this.params.excludeIDs) {
			this.params.excludeIDs = exclude;
		} else {
			this.params.excludeIDs.concat(exclude);
		}
	}

	//

	// setFilters(filters: Filter | Filter[]) {
	// 	this.options.filter = ensureArray(filters);
	// 	return this;
	// }

	// getFilterById(id: string): CompoundFilter | undefined {
	// 	return ensureArray(this.options.filter).find((f) => typeof f === 'object' && f.id == id) as
	// 		| CompoundFilter
	// 		| undefined;
	// }

	// addFilter(filter: string, id?: string, mode?: FilterMode) {
	// 	if (!id) {
	// 		this.options.filter = [...ensureArray(this.options.filter), filter];
	// 	} else {
	// 		const existingFilter = this.getFilterById(id);
	// 		if (existingFilter) {
	// 			existingFilter.expressions.push(filter);
	// 		} else {
	// 			this.options.filter = [
	// 				...ensureArray(this.options.filter),
	// 				{ id, expressions: [filter], mode: mode ?? '&&' }
	// 			];
	// 		}
	// 	}
	// 	return this;
	// }

	// removeFilter(expression: string, id?: string) {
	// 	if (!id) {
	// 		this.options.filter = ensureArray(this.options.filter).filter((f) => f !== expression);
	// 	} else {
	// 		const existingFilter = this.getFilterById(id);
	// 		if (!existingFilter) return this;
	// 		existingFilter.expressions = existingFilter.expressions.filter((e) => e !== expression);
	// 		if (existingFilter.expressions.length == 0) {
	// 			this.options.filter = ensureArray(this.options.filter).filter(
	// 				(f) => typeof f === 'object' && f.id !== id
	// 			);
	// 		}
	// 	}
	// 	return this;
	// }

	// hasFilter(expression: string, id?: string) {
	// 	if (!id) {
	// 		return ensureArray(this.options.filter).includes(expression);
	// 	} else {
	// 		return ensureArray(this.options.filter).some(
	// 			(f) => typeof f === 'object' && f.id == id && f.expressions.includes(expression)
	// 		);
	// 	}
	// }

	//

	//

	// //

	// addSort(field: Field<C>, order: SortOrder) {
	// 	this.options.sort = [...ensureSortOptionArray(this.options.sort), [field, order]];
	// 	return this;
	// }

	// setSort(field: Field<C>, order: SortOrder) {
	// 	this.options.sort = [[field, order]];
	// 	return this;
	// }

	// flipSort(sort: SortOption<Field<C>>) {
	// 	const sorts = ensureSortOptionArray(this.options.sort);
	// 	const sortToChange = sorts.find((s) => s[0] == sort[0] && s[1] == sort[1]);
	// 	if (!sortToChange) return this;
	// 	const index = sorts.indexOf(sortToChange);
	// 	sorts[index] = [sortToChange[0], sortToChange[1] == 'ASC' ? 'DESC' : 'ASC'];
	// 	this.options.sort = sorts;
	// 	return this;
	// }

	// hasSort(field: Field<C> | string) {
	// 	return ensureArray(this.getMergedOptions().sort).some((s) => s[0] == field);
	// }

	// getSort(field: Field<C> | string) {
	// 	return ensureArray(this.getMergedOptions().sort).find((s) => s[0] == field);
	// }
}
