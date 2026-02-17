<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Plus } from '@lucide/svelte';
	import { Pipeline } from '$lib';
	import { userOrganization } from '$lib/app-state';
	import { PolledResource } from '$lib/utils/state.svelte.js';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import PipelineCard from './_partials/pipeline-card.svelte';

	//

	let { data } = $props();

	setDashboardNavbar({ title: 'Pipelines', right: navbarRight });

	//

	const allWorkflows = new PolledResource(() => Pipeline.Workflows.listAllGroupedByPipelineId(), {
		initialValue: () => data.workflows,
		intervalMs: 10000
	});
</script>

<!-- Your Pipelines Section -->
<div class="mb-8">
	<T tag="h2" class="mb-4 text-lg font-semibold">{m.My()} {m.Pipelines()}</T>
	<CollectionManager
		collection="pipelines"
		queryOptions={{
			filter: `owner.id = '${userOrganization.current?.id}'`,
			sort: ['created', 'DESC'],
			expand: ['schedules_via_pipeline']
		}}
		hide={['pagination']}
	>
		{#snippet records({ records })}
			<div class="space-y-4">
				{#each records as pipeline, index (pipeline.id)}
					{@const workflows = allWorkflows.current?.[pipeline.id]}
					{#if userOrganization.current}
						<PipelineCard bind:pipeline={records[index]} {workflows} />
					{/if}
				{/each}
			</div>
		{/snippet}

		{#snippet emptyState({ EmptyState })}
			<EmptyState
				title={m.No_items_here()}
				description={m.Start_by_adding_a_record_to_this_collection_()}
			/>
		{/snippet}
	</CollectionManager>
</div>

<!-- Other Pipelines Section -->
<div>
	<T tag="h2" class="mb-4 text-lg font-semibold">{m.All()} {m.Pipelines()}</T>
	<CollectionManager
		collection="pipelines"
		queryOptions={{
			filter: `owner.id != '${userOrganization.current?.id}'`,
			sort: ['created', 'DESC'],
			expand: ['owner', 'schedules_via_pipeline']
		}}
		hide={['pagination', 'empty_state']}
	>
		{#snippet records({ records })}
			<div class="space-y-4">
				{#each records as pipeline, index (pipeline.id)}
					{@const ownerOrg = pipeline.expand?.owner}
					{@const workflows = allWorkflows.current?.[pipeline.id] ?? []}
					{#if ownerOrg}
						<PipelineCard
							bind:pipeline={records[index]}
							{workflows}
							onRun={() => allWorkflows.fetch()}
						/>
					{/if}
				{/each}
			</div>
		{/snippet}
	</CollectionManager>
</div>

{#snippet navbarRight()}
	<Button href="/my/pipelines/new">
		<Plus />
		{m.New()}
	</Button>
{/snippet}
