<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { userOrganization } from '$lib/app-state';
	import EmptyState from '$lib/pipeline-form/steps/_partials/empty-state.svelte';
	import ItemCard from '$lib/pipeline-form/steps/_partials/item-card.svelte';
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
	import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';
	import { getPath } from '$lib/utils';

	import type { MobileRunnersResponse } from '@/pocketbase/types';

	import { Badge } from '@/components/ui/badge';
	import Label from '@/components/ui/label/label.svelte';
	import { cn } from '@/components/ui/utils';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import * as status from './status.svelte.js';

	//

	type Props = {
		onSelect?: (runner: MobileRunnersResponse) => void;
		selectedRunner?: string;
		name?: string;
		required?: boolean;
	};

	let { onSelect, selectedRunner, name, required = false }: Props = $props();

	//

	let foundRunners = $state<MobileRunnersResponse[]>([]);

	const runnerSearch = new Search({
		onSearch: (text) => {
			searchRunner(text);
		}
	});

	async function searchRunner(text: string) {
		const filter = pb.filter(
			[
				['name ~ {:text}', 'canonified_name ~ {:text}'].join(' || '),
				['owner.id = {:currentOrganization}', 'published = true'].join(' || ')
			]
				.map((f) => `(${f})`)
				.join(' && '),
			{
				text: text,
				currentOrganization: userOrganization.current?.id
			}
		);
		foundRunners = await pb.collection('mobile_runners').getFullList({
			requestKey: null,
			filter: filter,
			sort: 'created'
		});
	}

	$effect(() => {
		const runners = foundRunners;
		if (runners.length === 0) return;
		status.probe(runners, { reason: 'visible' });
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
		{#each foundRunners as item (item.id)}
			{@const isSelected = selectedRunner === getPath(item)}
			{@const runnerPath = getPath(item)}
			{@const online = status.isOnline(runnerPath)}
			<ItemCard
				title={item.name}
				onClick={(e) => {
					e.preventDefault();
					onSelect?.(item);
				}}
				class={isSelected ? 'border-blue-500 bg-blue-50!' : ''}
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
							title={online === true
								? 'Online'
								: online === false
									? 'Offline'
									: 'Checking status'}
						></span>
						{#if !item.published}
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
