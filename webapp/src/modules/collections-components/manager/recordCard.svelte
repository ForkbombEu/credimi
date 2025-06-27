<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { cn } from '@/components/ui/utils';
	import type { CollectionResponses } from '@/pocketbase/types';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import ItemCard from '@/components/ui-custom/itemCard.svelte';
	import { getCollectionManagerContext } from './collectionManagerContext';
	import {
		RecordSelect,
		type HideOption,
		RecordEdit,
		RecordShare,
		RecordDelete
	} from './record-actions';
	import type { Snippet } from 'svelte';
	import type ItemCardTitle from '@/components/ui-custom/itemCardTitle.svelte';
	import type ItemCardDescription from '@/components/ui-custom/itemCardDescription.svelte';

	interface Props {
		record: CollectionResponses[C];
		hide?: HideOption;
		class?: string;
		children?: Snippet<
			[{ Title: typeof ItemCardTitle; Description: typeof ItemCardDescription }]
		>;
		right?: Snippet<[{ record: CollectionResponses[C] }]>;
	}

	let {
		record,
		hide = [],
		class: className = '',
		children: children_render,
		right: right_render
	}: Props = $props();

	//

	const { manager } = $derived(getCollectionManagerContext());

	const classes = $derived(
		cn(className, {
			'border-primary': manager.selectedRecords.includes(record.id)
		})
	);
</script>

<ItemCard class="{classes} " left={!hide.includes('select') && hide !== 'all' ? left : undefined}>
	{#snippet children({ Title, Description })}
		{@render children_render?.({ Title, Description })}
	{/snippet}

	{#snippet right()}
		{@render right_render?.({ record })}

		{#if !hide.includes('edit') && hide !== 'all'}
			<RecordEdit {record} />
		{/if}
		{#if !hide.includes('share') && hide !== 'all'}
			<RecordShare {record} />
		{/if}
		{#if !hide.includes('delete') && hide !== 'all'}
			<RecordDelete {record} />
		{/if}
	{/snippet}
</ItemCard>

{#snippet left()}
	<RecordSelect {record} />
{/snippet}
