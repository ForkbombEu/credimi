<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script
	lang="ts"
	generics="C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { Snippet } from 'svelte';

	import { FolderIcon, MessageCircleWarning, SearchIcon } from '@lucide/svelte';

	import type { FormOptions as SuperformsOptions } from '@/forms';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type {
		PocketbaseQueryAgentOptions,
		PocketbaseQueryExpandOption,
		PocketbaseQueryOptions,
		PocketbaseQueryResponse
	} from '@/pocketbase/query';
	import type { CollectionFormData, CollectionResponses } from '@/pocketbase/types';
	import type { CollectionZodSchema } from '@/pocketbase/zod-schema';

	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	import { m } from '@/i18n';
	import { setupComponentPocketbaseSubscriptions } from '@/pocketbase/subscriptions';

	import type {
		UIOptions as CollectionFormUIOptions,
		FieldsOptions
	} from '../form/collectionFormTypes';

	import { CollectionManager } from './collectionManager.svelte.js';
	import {
		setCollectionManagerContext,
		type CollectionManagerContext,
		type FiltersOption
	} from './collectionManagerContext';
	import Filters from './collectionManagerFilters.svelte';
	import Header from './collectionManagerHeader.svelte';
	import Pagination from './collectionManagerPagination.svelte';
	import Search from './collectionManagerSearch.svelte';
	import EditForm from './forms/edit-form.svelte';
	import Card from './recordCard.svelte';
	import Table from './table/collectionTable.svelte';

	//

	type Props = {
		collection: C;
		onMount?: (manager: CollectionManager<C, E>) => void;
	} & Partial<Options> &
		Partial<Snippets>;

	type Snippets = {
		createForm: Snippet<[{ closeSheet: () => void }]>;
		editForm: Snippet<[{ record: CollectionResponses[C]; closeSheet: () => void }]>;

		records: Snippet<
			[
				{
					records: PocketbaseQueryResponse<C, E>[];
					Card: typeof Card;
					Table: typeof Table;
					Pagination: typeof Pagination;
					totalRecords: number;
					reloadRecords: () => void;
					pageRange: string;
				}
			]
		>;
		emptyState: Snippet<[{ EmptyState: typeof EmptyState }]>;
		top: Snippet<
			[
				{
					Search: typeof Search;
					Header: typeof Header;
					Filters: typeof Filters;
					records: PocketbaseQueryResponse<C, E>[];
					totalRecords: number;
					reloadRecords: () => void;
					pageRange: string;
				}
			]
		>;
		contentWrapper: Snippet<[children: () => ReturnType<Snippet>]>;
	};

	type Options = {
		queryOptions: PocketbaseQueryOptions<C, E>;
		queryAgentOptions: PocketbaseQueryAgentOptions;
		filters: FiltersOption;
		subscribe: 'off' | 'expanded_collections' | CollectionName[];

		hide: ('empty_state' | 'pagination')[];

		formRefineSchema: (schema: CollectionZodSchema<C>) => CollectionZodSchema<C>;

		formUIOptions: CollectionFormUIOptions;
		createFormUIOptions: CollectionFormUIOptions;
		editFormUIOptions: CollectionFormUIOptions;

		formSuperformsOptions: SuperformsOptions<CollectionFormData[C]>;
		createFormSuperformsOptions: SuperformsOptions<CollectionFormData[C]>;
		editFormSuperformsOptions: SuperformsOptions<CollectionFormData[C]>;

		formFieldsOptions: Partial<FieldsOptions<C>>;
		createFormFieldsOptions: Partial<FieldsOptions<C>>;
		editFormFieldsOptions: Partial<FieldsOptions<C>>;

		emptyStateTitle: string;
		emptyStateDescription: string;
	};

	//

	const {
		collection,
		queryOptions = {},
		queryAgentOptions = {},
		hide = [],
		top,
		records,
		emptyState,
		contentWrapper,
		filters = [],
		emptyStateTitle = m.No_items_here(),
		emptyStateDescription = m.Start_by_adding_a_record_to_this_collection_(),
		onMount,
		...rest
	}: Props = $props();
	
	//

	const manager = new CollectionManager(collection, {
		query: () => queryOptions,
		queryAgent: queryAgentOptions
	});

	$effect(() => {
		onMount?.(manager);
	});

	const context = $derived<CollectionManagerContext<C, E>>({
		manager,
		filters,
		formsOptions: {
			base: {
				uiOptions: rest.formUIOptions,
				superformsOptions: rest.formSuperformsOptions,
				fieldsOptions: rest.formFieldsOptions
			},
			create: {
				uiOptions: rest.createFormUIOptions,
				superformsOptions: rest.createFormSuperformsOptions,
				fieldsOptions: rest.createFormFieldsOptions
			},
			edit: {
				uiOptions: rest.editFormUIOptions,
				superformsOptions: rest.editFormSuperformsOptions,
				fieldsOptions: rest.editFormFieldsOptions
			}
		},
		formRefineSchema: rest.formRefineSchema ?? ((schema) => schema),
		createForm: rest.createForm,
		editForm: rest.editForm
	});

	setCollectionManagerContext(() => context);

	setupComponentPocketbaseSubscriptions({
		collection,
		callback: () => manager.loadRecords(),
		expandOption: queryOptions.expand,
		other: ['authorizations']
	});
</script>

{@render top?.({
	Search,
	Header,
	Filters,
	records: manager.records,
	totalRecords: manager.totalItems,
	pageRange: manager.currentRange,
	reloadRecords: () => {
		manager.loadRecords();
	}
})}
{@render (contentWrapper ?? defaultContentWrapper)(content)}

{#snippet defaultContentWrapper(children: () => ReturnType<Snippet>)}
	{@render children()}
{/snippet}

{#snippet content()}
	{#if manager.loadingError}
		<EmptyState
			title={m.Error()}
			description={m.Some_error_occurred_while_loading_records_()}
			icon={MessageCircleWarning}
		/>
	{:else if manager.records.length > 0}
		{@render records?.({
			records: manager.records,
			Card,
			Table,
			Pagination,
			totalRecords: manager.totalItems,
			pageRange: manager.currentRange,
			reloadRecords: () => {
				manager.loadRecords();
			}
		})}

		{#if !hide.includes('pagination')}
			<Pagination class="mt-6" />
		{/if}
	{:else if manager.query.hasSearch() && manager.records.length === 0}
		<EmptyState title={m.No_records_found()} icon={SearchIcon} />
	{:else if emptyState}
		{@render emptyState({ EmptyState })}
	{:else if !hide.includes('empty_state')}
		<EmptyState title={emptyStateTitle} description={emptyStateDescription} icon={FolderIcon} />
	{/if}
{/snippet}

<EditForm />
