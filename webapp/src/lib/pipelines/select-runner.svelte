<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';
	import type { SelfProp } from '$lib/renderable';

	import { userOrganization } from '$lib/app-state';
	import { pb } from '@/pocketbase';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import ItemCard from '$lib/pipeline-form/steps/_partials/item-card.svelte';
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
	import WithEmptyState from '$lib/pipeline-form/steps/_partials/with-empty-state.svelte';
	import WithLabel from '$lib/pipeline-form/steps/_partials/with-label.svelte';
	import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';

	//

	type Props = {
		pipeline: PipelinesResponse;
		onRunnerSelect: (runner: MobileRunnersResponse) => void;
	};

	let { pipeline, onRunnerSelect }: Props = $props();

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

	// Initialize search with empty string to load all runners
	let initialized = false;
	$effect(() => {
		if (!initialized) {
			initialized = true;
			searchRunner('');
		}
	});
</script>

<WithLabel label={m.Runner()} class="p-4">
	<SearchInput search={runnerSearch} />
</WithLabel>

<WithEmptyState items={foundRunners} emptyText={m.No_runners_found()}>
	{#snippet item({ item })}
		<ItemCard title={item.name} onClick={() => onRunnerSelect(item)}>
			{#snippet right()}
				{#if !item.published}
					<Badge variant="secondary">
						{m.private()}
					</Badge>
				{/if}
			{/snippet}
		</ItemCard>
	{/snippet}
</WithEmptyState>
