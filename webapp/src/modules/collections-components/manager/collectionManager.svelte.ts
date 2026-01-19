// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ClientResponseError, RecordService } from 'pocketbase';

import { Array } from 'effect';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { RecordIdString } from '@/pocketbase/types';

import { pb } from '@/pocketbase';
import {
	type PocketbaseQueryAgentOptions,
	type PocketbaseQueryExpandOption,
	type PocketbaseQueryOptions,
	type PocketbaseQueryResponse,
	PocketbaseQueryAgent,
	PocketbaseQueryOptionsEditor
} from '@/pocketbase/query';

import type { RecordEditProps } from './record-actions/types';

//

export class CollectionManager<
	C extends CollectionName,
	E extends PocketbaseQueryExpandOption<C> = never
> {
	recordService: RecordService<PocketbaseQueryResponse<C, E>>;

	private rootQueryOptions: PocketbaseQueryOptions<C, E> = $state({});
	private currentQueryOptions: PocketbaseQueryOptions<C, E> = $state({});
	query = $derived.by(
		() => new PocketbaseQueryOptionsEditor(this.currentQueryOptions, this.rootQueryOptions)
	);

	private queryAgentOptions: PocketbaseQueryAgentOptions = $state({});
	private queryAgent = $derived.by(
		() =>
			new PocketbaseQueryAgent(
				{
					collection: this.collection,
					...this.query.getMergedOptions()
				},
				{ ...this.queryAgentOptions, requestKey: null }
			)
	);

	constructor(
		public readonly collection: C,
		options: {
			query: PocketbaseQueryOptions<C, E>;
			queryAgent: PocketbaseQueryAgentOptions;
		}
	) {
		this.rootQueryOptions = options.query;
		this.queryAgentOptions = options.queryAgent;
		this.recordService = pb.collection(collection);

		$effect(() => {
			this.loadRecords();
		});
	}

	/* Data loading */

	records = $state<PocketbaseQueryResponse<C, E>[]>([]);
	currentPage = $state(1);
	totalItems = $state(0);
	loadingError = $state<ClientResponseError>();
	currentRange = $derived.by(() => {
		if (!this.query.hasPagination()) return `1 - ${this.records.length}`;
		else {
			const pageSize = this.query.getPageSize() ?? 1;
			const start = 1 + (this.currentPage - 1) * pageSize;
			const end = Math.min(this.currentPage * pageSize, this.totalItems);
			return `${start} - ${end}`;
		}
	});

	private previousFilter: string | undefined;

	async loadRecords() {
		const currentFilter = this.queryAgent.listOptions.filter;
		if (this.previousFilter !== currentFilter) {
			this.currentPage = 1;
			this.previousFilter = currentFilter;
		}

		try {
			if (this.query.hasPagination()) {
				const result = await this.queryAgent.getList(this.currentPage);
				this.records = result.items;
				this.totalItems = result.totalItems;
			} else {
				this.records = await this.queryAgent.getFullList();
				this.totalItems = this.records.length;
			}
		} catch (e) {
			console.error(e);
			this.loadingError = e as ClientResponseError;
		}
	}

	/* Selection */

	selectedRecords = $state<RecordIdString[]>([]);

	areAllRecordsSelected() {
		return this.records.every((r) => this.selectedRecords.includes(r.id));
	}

	toggleSelectAllRecords() {
		const allSelected = this.areAllRecordsSelected();
		if (allSelected) {
			this.selectedRecords = [];
		} else {
			this.selectedRecords = this.records.map((r) => r.id);
		}
	}

	discardSelection() {
		this.selectedRecords = [];
	}

	selectRecord(id: RecordIdString) {
		this.selectedRecords.push(id);
	}

	deselectRecord(id: RecordIdString) {
		this.selectedRecords = Array.remove(this.selectedRecords, this.selectedRecords.indexOf(id));
	}

	/* Forms */

	isEditFormOpen = $state(false);
	editFormProps = $state<RecordEditProps<C>>();

	openEditForm(props: RecordEditProps<C>) {
		this.isEditFormOpen = true;
		this.editFormProps = props;
	}

	closeEditForm() {
		this.isEditFormOpen = false;
		this.editFormProps = undefined;
	}
}
