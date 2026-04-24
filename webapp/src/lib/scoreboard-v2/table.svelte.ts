// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getCoreRowModel, type PaginationState, type Table } from '@tanstack/table-core';
import { onMount } from 'svelte';

import { createSvelteTable } from '@/components/ui/data-table';

import type { ScoreboardRow } from './types';

import * as Column from './column';
import * as conformanceChecks from './columns/conformance-checks.svelte';
import * as credentials from './columns/credentials.svelte';
import * as customIntegrations from './columns/custom-integrations.svelte';
import * as issuers from './columns/issuers.svelte';
import * as minimumRunningTime from './columns/minimum-running-time.svelte';
import * as name from './columns/name.svelte';
import * as runners from './columns/runners.svelte';
import * as totalExecutionsSuccessesPercentage from './columns/total-executions-successes-percentage.svelte';
import * as useCaseVerifications from './columns/use-case-verifications.svelte';
import * as verifiers from './columns/verifiers.svelte';
import * as videoScreenshot from './columns/video-screenshot.svelte';
import * as wallets from './columns/wallets.svelte';
import { loadScoreboardData } from './functions';

//

const columns = [
	Column.build(name),
	Column.build(videoScreenshot),
	Column.build(totalExecutionsSuccessesPercentage),
	Column.build(wallets),
	Column.build(issuers),
	Column.build(credentials),
	Column.build(verifiers),
	Column.build(useCaseVerifications),
	Column.build(conformanceChecks),
	Column.build(customIntegrations),
	Column.build(runners),
	Column.build(minimumRunningTime)
];

interface ExtendedPaginationState extends PaginationState {
	totalItems: number;
	pageCount: number;
}

interface Options {
	pageSize?: number;
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

	get pageSize() {
		return this.#pagination.pageSize;
	}
	get pageCount() {
		return this.#pagination.pageCount;
	}
	get totalItems() {
		return this.#pagination.totalItems;
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

		this.table = createSvelteTable({
			columns,
			getCoreRowModel: getCoreRowModel(),
			get data() {
				return getData();
			},
			state: {
				get pagination() {
					return getPagination();
				}
			},
			onPaginationChange: (updater) => {
				setPagination(typeof updater === 'function' ? updater(getPagination()) : updater);
				this.loadData();
			},
			manualPagination: true,
			get pageCount() {
				return getPageCount();
			}
		});

		onMount(() => this.loadData());
	}

	private async loadData() {
		const currentApiPage = toApiPage(this.currentPage);
		const res = await loadScoreboardData({
			pagination: {
				page: currentApiPage,
				perPage: this.#pagination.pageSize
			}
		});
		const normalizedApiPage = fromApiPage(res.page);
		this.#data = res.items;
		this.#pagination = {
			pageSize: res.perPage,
			pageIndex: toTableIndex(normalizedApiPage),
			pageCount: res.totalPages,
			totalItems: res.totalItems
		};
	}
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
