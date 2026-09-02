// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListResult } from 'pocketbase';

import {
	getCoreRowModel,
	type PaginationState,
	type SortingState,
	type Table
} from '@tanstack/table-core';
import { onMount } from 'svelte';

import { createSvelteTable } from '@/components/ui/data-table';

import type { ScoreboardRow } from './types';

import * as Column from './column';
import * as conformanceChecks from './columns/conformance-checks.svelte';
import * as issuers from './columns/issuers.svelte';
import * as lastExecution from './columns/last-execution.svelte';
import * as name from './columns/name.svelte';
import * as runners from './columns/runners.svelte';
import * as verifiers from './columns/verifiers.svelte';
import * as videoScreenshot from './columns/video-screenshot.svelte';
import * as wallets from './columns/wallets.svelte';
import { loadPage } from './records';

//

const columns = [
	Column.build(name),
	Column.build(videoScreenshot),
	Column.build(wallets),
	Column.build(issuers),
	Column.build(verifiers),
	Column.build(conformanceChecks),
	Column.build(runners),
	Column.build(lastExecution)
];

interface ExtendedPaginationState extends PaginationState {
	totalItems: number;
	pageCount: number;
}

interface Options {
	pageSize?: number;
	initialPage?: () => ListResult<ScoreboardRow>;
}

export class ScoreboardTable {
	public readonly table: Table<ScoreboardRow>;

	#data = $state<ScoreboardRow[]>([]);

	#pagination = $state<ExtendedPaginationState>({
		pageIndex: 0,
		pageSize: 5,
		totalItems: 0,
		pageCount: 0
	});

	#sorting = $state<SortingState>([{ id: lastExecution.column.id, desc: true }]);

	#filter = $state<string | undefined>(undefined);

	#columnVisibility = $state<Record<string, boolean>>({});

	get pageSize() {
		return this.#pagination.pageSize;
	}
	get pageCount() {
		return this.#pagination.pageCount;
	}
	get totalItems() {
		return this.#pagination.totalItems;
	}

	get filter() {
		return this.#filter;
	}

	setFilter(filter: string | undefined) {
		this.#filter = filter;
		this.#pagination.pageIndex = 0;
		this.loadData();
	}

	get columnVisibility() {
		return this.#columnVisibility;
	}

	toggleColumn(id: string, visible: boolean) {
		this.#columnVisibility = { ...this.#columnVisibility, [id]: visible };
	}

	get currentPage() {
		return fromTableIndex(this.#pagination.pageIndex);
	}
	set currentPage(page: number) {
		this.table.setPageIndex(toTableIndex(page));
	}

	constructor(options: Options = {}) {
		const { pageSize = 5 } = options;
		this.#pagination.pageSize = pageSize;

		const getData = () => this.#data;
		const getPagination = () => this.#pagination;
		const getPageCount = () => this.#pagination.pageCount;
		const setPagination = (p: PaginationState) => {
			this.#pagination.pageIndex = p.pageIndex;
			this.#pagination.pageSize = p.pageSize;
		};
		const getSorting = () => this.#sorting;
		const getColumnVisibility = () => this.#columnVisibility;

		this.table = createSvelteTable({
			columns,
			getCoreRowModel: getCoreRowModel(),
			get data() {
				return getData();
			},
			state: {
				get pagination() {
					return getPagination();
				},
				get sorting() {
					return getSorting();
				},
				get columnVisibility() {
					return getColumnVisibility();
				}
			},
			onPaginationChange: (updater) => {
				setPagination(typeof updater === 'function' ? updater(getPagination()) : updater);
				this.loadData();
			},
			onSortingChange: (updater) => {
				const next = typeof updater === 'function' ? updater(getSorting()) : updater;
				this.#sorting = next;
				this.#pagination.pageIndex = 0;
				this.loadData();
			},
			onColumnVisibilityChange: (updater) => {
				const next =
					typeof updater === 'function' ? updater(getColumnVisibility()) : updater;
				this.#columnVisibility = next;
			},
			manualPagination: true,
			manualSorting: true,
			get pageCount() {
				return getPageCount();
			}
		});

		onMount(() => {
			if (!options.initialPage) this.loadData();
		});

		$effect(() => {
			if (options.initialPage) {
				this.applyPageResult(options.initialPage());
			}
		});
	}

	private applyPageResult(res: ListResult<ScoreboardRow>) {
		const normalizedApiPage = fromApiPage(res.page);
		this.#data = res.items;
		this.#pagination = {
			pageSize: res.perPage,
			pageIndex: toTableIndex(normalizedApiPage),
			pageCount: res.totalPages,
			totalItems: res.totalItems
		};
	}

	private async loadData() {
		const currentApiPage = toApiPage(this.currentPage);
		const sort = buildSortString(this.table, this.#sorting);
		const res = await loadPage({
			page: currentApiPage,
			perPage: this.#pagination.pageSize,
			...(sort ? { sort } : {}),
			...(this.#filter ? { filter: this.#filter } : {})
		});
		this.applyPageResult(res);
	}
}

function buildSortString(table: Table<ScoreboardRow>, sorting: SortingState): string {
	return sorting
		.map(({ id, desc }) => {
			const field = table.getColumn(id)?.columnDef.meta?.sortField;
			return field ? `${desc ? '-' : '+'}${field}` : null;
		})
		.filter((s): s is string => Boolean(s))
		.join(',');
}

// helpers to convert between table and API pagination

function fromTableIndex(index0: number) {
	return Math.max(0, index0) + 1;
}

function toTableIndex(page1: number) {
	return Math.max(0, page1 - 1);
}

function fromApiPage(page1: number) {
	return Math.max(1, page1);
}

function toApiPage(page1: number) {
	return Math.max(1, page1);
}
