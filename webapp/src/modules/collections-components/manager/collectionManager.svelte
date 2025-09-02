<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script
	lang="ts"
	generics="C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { Snippet } from 'svelte';

	import { FolderIcon, MessageCircleWarning, SearchIcon } from 'lucide-svelte';

	// Logic - Types
	import type { FormOptions as SuperformsOptions } from '@/forms';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type {
		PocketbaseQueryAgentOptions,
		PocketbaseQueryExpandOption,
		PocketbaseQueryOptions,
		PocketbaseQueryResponse
	} from '@/pocketbase/query';
	import type { CollectionFormData } from '@/pocketbase/types';

	// Components
	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	// UI
	import { m } from '@/i18n';
	// Logic
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
	import Card from './recordCard.svelte';
	import Table from './table/collectionTable.svelte';

	//

	type Props = {
		collection: C;
	} & Partial<Options> &
		Partial<Snippets>;

	type Snippets = {
		records: Snippet<
			[
				{
					records: PocketbaseQueryResponse<C, E>[];
					Card: typeof Card;
					Table: typeof Table;
					Pagination: typeof Pagination;
					reloadRecords: () => void;
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
					reloadRecords: () => void;
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
		subscribe = 'expanded_collections',
		top,
		records,
		emptyState,
		contentWrapper,
		filters = [],
		emptyStateTitle = m.No_items_here(),
		emptyStateDescription = m.Start_by_adding_a_record_to_this_collection_(),
		...rest
	}: Props = $props();
	//

	const manager = $derived(
		new CollectionManager(collection, {
			query: queryOptions,
			queryAgent: queryAgentOptions
		})
	);

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
		}
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
