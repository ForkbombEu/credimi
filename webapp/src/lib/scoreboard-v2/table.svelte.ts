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
import * as wallets from './columns/wallets.svelte';
import { loadScoreboardData } from './functions';

//

const columns = [
	Column.build(name),
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

export class ScoreboardTable {
	#data = $state<ScoreboardRow[]>([]);
	#pagination = $state<PaginationState>({ pageIndex: 0, pageSize: 10 });

	public readonly table: Table<ScoreboardRow>;

	constructor() {
		const getData = () => this.#data;
		const getPagination = () => this.#pagination;

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
			}
		});

		onMount(async () => {
			this.#data = await loadScoreboardData();
		});
	}
}
