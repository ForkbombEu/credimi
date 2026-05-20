<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import EmptyState from '$lib/pipeline-form/steps/_partials/empty-state.svelte';
	import ItemCard from '$lib/pipeline-form/steps/_partials/item-card.svelte';
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
	import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';

	import { Pipeline } from '$lib';
	import type { Record } from '$lib/pipeline/runner';

	import { Badge } from '@/components/ui/badge';
	import Label from '@/components/ui/label/label.svelte';
	import { cn } from '@/components/ui/utils';
	import { m } from '@/i18n';

	type Props = {
		onSelect?: (runner: Record) => void;
		selectedRunner?: string;
		name?: string;
		required?: boolean;
	};

	let { onSelect, selectedRunner, name, required = false }: Props = $props();

	const runnerSearch = new Search({
		onSearch: (text) => {
			searchRunner(text);
		}
	});

	function searchRunner(_text: string) {}

	const foundRunners = $derived.by(() => {
		Pipeline.Runner.Catalog.read();
		return Pipeline.Runner.Catalog.search(runnerSearch.text);
	});
</script>

<div class="space-y-3">
	<div class="space-y-2">
		<Label for={name}>
			{m.Runner()}
			{#if required}
				<span class="font-bold text-destructive">*</span>
			{/if}
		</Label>
		<SearchInput search={runnerSearch} {name} />
	</div>

	<div class="space-y-1">
		{#each foundRunners as item (item.path)}
			{@const isSelected = selectedRunner === item.path}
			{@const online = !Pipeline.Runner.Catalog.isReady() ? undefined : item.isOnline}
			{@const isOffline = online === false}
			<ItemCard
				title={item.name}
				onClick={isOffline
					? undefined
					: (e) => {
							e.preventDefault();
							onSelect?.(item);
						}}
				class={cn(
					isSelected && 'border-blue-500 bg-blue-50!',
					isOffline && 'cursor-not-allowed bg-slate-100! opacity-50'
				)}
				tooltip={isOffline ? m.Runner_offline_select_disabled() : undefined}
			>
				{#snippet afterContent()}
					<div class="text-xs text-balance text-muted-foreground">{item.description}</div>
				{/snippet}

				{#snippet right()}
					<div class="flex items-center gap-2">
						<span
							class={cn(
								'size-2 shrink-0 rounded-full',
								online === true && 'bg-green-500',
								online === false && 'bg-red-400',
								online === undefined && 'bg-muted-foreground/40'
							)}
							title={online === undefined
								? m.Runner_status_checking()
								: online === true
									? 'Online'
									: 'Offline'}
						></span>
						{#if !item.isPublished}
							<Badge variant="secondary">
								{m.private()}
							</Badge>
						{/if}
					</div>
				{/snippet}
			</ItemCard>
		{:else}
			<EmptyState text={m.No_runners_found()} />
		{/each}
	</div>
</div>
