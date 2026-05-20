<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import EmptyState from '$lib/pipeline-form/steps/_partials/empty-state.svelte';
	import ItemCard from '$lib/pipeline-form/steps/_partials/item-card.svelte';
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
	import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';

	import { Badge } from '@/components/ui/badge';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n';

	import type { MobileRunnerListItem } from '../runners/utils';

	import * as Runners from '../runners';
	import * as status from '../runners/status.svelte.js';

	//

	type Props = {
		onSelect?: (runner: MobileRunnerListItem) => void;
		selectedRunner?: string;
		name?: string;
		required?: boolean;
	};

	let { onSelect, selectedRunner, name, required = false }: Props = $props();

	//

	let foundRunners = $state<MobileRunnerListItem[]>([]);

	const runnerSearch = new Search({
		onSearch: (text) => {
			searchRunner(text);
		}
	});

	function searchRunner(text: string) {
		const search = text.trim().toLowerCase();
		foundRunners = Runners.store.read().filter((runner) => {
			if (!search) return true;
			return (
				runner.name.toLowerCase().includes(search) ||
				runner.runner_id.toLowerCase().includes(search)
			);
		});
	}

	$effect(() => {
		const runners = foundRunners;
		if (runners.length === 0) return;
		status.probe(runners, { reason: 'visible' });
	});

	$effect(() => {
		Runners.store.read();
		searchRunner(runnerSearch.text);
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
		{#each foundRunners as item (item.runner_id)}
			{@const isSelected = selectedRunner === item.runner_id}
			{@const runnerPath = item.runner_id}
			{@const online = status.isOnline(runnerPath)}
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
					<div class="text-xs text-balance text-muted-foreground">{item.description}</div>
				{/snippet}

				{#snippet right()}
					<div class="flex items-center gap-2">
						{#if !item.published}
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
							title={online === true
								? 'Online'
								: online === false
									? 'Offline'
									: 'Checking status'}
						></span>
					</div>
				{/snippet}
			</ItemCard>
		{:else}
			<EmptyState text={m.No_runners_found()} />
		{/each}
	</div>
</div>
