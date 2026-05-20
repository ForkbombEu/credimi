<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import EmptyState from '$lib/pipeline-form/steps/_partials/empty-state.svelte';
	import ItemCard from '$lib/pipeline-form/steps/_partials/item-card.svelte';
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
	import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';
	import { fly } from 'svelte/transition';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		onSelect?: (runner: Pipeline.Runner.Record) => void;
		selectedRunner?: string;
		name?: string;
		required?: boolean;
	};

	let { onSelect, selectedRunner, name, required = false }: Props = $props();

	//

	let foundRunners = $state<Pipeline.Runner.Record[]>([]);

	const catalogLoading = $derived(!Pipeline.Runner.Catalog.isReady());

	const runnerSearch = new Search({
		onSearch: () => {
			foundRunners = Pipeline.Runner.Catalog.search(runnerSearch.text);
		}
	});

	$effect(() => {
		void Pipeline.Runner.Catalog.isReady();
		void Pipeline.Runner.Catalog.read();
		foundRunners = Pipeline.Runner.Catalog.search(runnerSearch.text);
	});
</script>

{#if catalogLoading}
	<EmptyState containerClass="p-0!">
		<Spinner size={16} />
		<T>{m.Loading()}</T>
	</EmptyState>
{:else}
	<div class="space-y-3" transition:fly>
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
					class={[
						'hover:cursor-pointer',
						isSelected && 'border-blue-500 bg-blue-50!',
						isOffline && 'bg-slate-100! opacity-50 hover:cursor-not-allowed'
					]}
					tooltip={isOffline ? m.Runner_offline_select_disabled() : undefined}
					hideArrow
				>
					{#snippet afterContent()}
						<div class="text-xs text-balance text-muted-foreground">
							{item.description}
						</div>
					{/snippet}

					{#snippet titleRight()}
						{#if isSelected}
							<Badge
								variant="secondary"
								class="-translate-y-px border border-blue-600 bg-blue-100 px-1 py-0 text-[10px] text-blue-600	"
							>
								{m.selected()}
							</Badge>
						{/if}
					{/snippet}

					{#snippet right()}
						<div class="flex items-center gap-2">
							{#if !item.isPublished}
								<Badge variant="secondary">
									{m.private()}
								</Badge>
							{/if}

							<span
								class={[
									'size-2 shrink-0 rounded-full border',
									online === true && 'border-emerald-500 bg-emerald-100',
									online === false && 'border-red-500 bg-red-100',
									online === undefined &&
										'border-muted-foreground/40 bg-muted-foreground/40'
								]}
								title={online === undefined
									? m.Runner_status_checking()
									: online === true
										? 'Online'
										: 'Offline'}
							></span>
						</div>
					{/snippet}
				</ItemCard>
			{:else}
				<EmptyState text={m.No_runners_found()} containerClass="p-0!" />
			{/each}
		</div>
	</div>
{/if}
