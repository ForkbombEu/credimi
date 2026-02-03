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
	import WithLabel from '$lib/pipeline-form/steps/_partials/with-label.svelte';

	import type { MobileRunnersResponse } from '@/pocketbase/types';

	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		onSelect?: (runner: MobileRunnersResponse) => void;
	};

	let { onSelect }: Props = $props();

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
</script>

<WithLabel label={m.Runner()}>
	<SearchInput search={runnerSearch} />
</WithLabel>

<div class="space-y-1">
	{#each foundRunners as item (item.id)}
		<ItemCard title={item.name} onClick={() => onSelect?.(item)}>
			{#snippet right()}
				{#if !item.published}
					<Badge variant="secondary">
						{m.private()}
					</Badge>
				{/if}
			{/snippet}
		</ItemCard>
	{:else}
		<EmptyState text={m.No_runners_found()} />
	{/each}
</div>
