<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script
	lang="ts"
	generics="C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import type { CollectionName } from '@/pocketbase/collections-models';
	import type {
		PocketbaseQueryAgentOptions,
		PocketbaseQueryExpandOption,
		PocketbaseQueryOptions,
		PocketbaseQueryResponse
	} from '@/pocketbase/query';
	import type { CollectionResponses } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import { ScrollArea } from '@/components/ui/scroll-area';
	import { m } from '@/i18n';

	import ItemCard from './item-card.svelte';
	import StepCollectionPickerPagination from './step-collection-picker-pagination.svelte';
	import WithLabel from './with-label.svelte';

	type Props = {
		collection: C;
		queryOptions: PocketbaseQueryOptions<C, E>;
		queryAgentOptions?: PocketbaseQueryAgentOptions;
		onSelect: (record: CollectionResponses[C]) => void;
		label?: string;
		class?: ClassValue;
		emptyText?: string;
		prepend?: Snippet;
		item?: Snippet<
			[
				{
					record: PocketbaseQueryResponse<C, E>;
					onSelect: (record: PocketbaseQueryResponse<C, E>) => void;
				}
			]
		>;
	};

	let {
		collection,
		queryOptions,
		queryAgentOptions = {},
		onSelect,
		label,
		class: className,
		emptyText,
		prepend,
		item: itemSnippet
	}: Props = $props();

	function handleSelect(record: PocketbaseQueryResponse<C, E>) {
		onSelect(record as CollectionResponses[C]);
	}
</script>

<div class={['flex min-h-0 flex-col', className]}>
	<CollectionManager
		{collection}
		queryOptions={{ perPage: 10, ...queryOptions }}
		{queryAgentOptions}
		hide={['empty_state', 'pagination']}
	>
		{#snippet top({ Search })}
			{#if label}
				<WithLabel {label} class="p-4">
					<Search />
				</WithLabel>
			{:else}
				<div class="p-4">
					<Search />
				</div>
			{/if}
			{@render prepend?.()}
		{/snippet}

		{#snippet records({ records, manager })}
			<StepCollectionPickerPagination />
			<ScrollArea
				class={[
					'grow [&>div>div]:space-y-2 [&>div>div]:p-4',
					manager.showPagination && '[&>div>div]:pt-0.5!'
				]}
			>
				{#each records as record (record.id)}
					{#if itemSnippet}
						{@render itemSnippet({ record, onSelect: handleSelect })}
					{:else}
						<ItemCard
							title={'name' in record && record.name
								? String(record.name)
								: record.id}
							onClick={() => handleSelect(record)}
						/>
					{/if}
				{/each}
			</ScrollArea>
		{/snippet}

		{#snippet emptyState({ EmptyState })}
			<div class="p-4">
				<EmptyState title={emptyText ?? m.No_items_here()} />
			</div>
		{/snippet}
	</CollectionManager>
</div>
