<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { PlayIcon, Plus } from '@lucide/svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { PolledResource } from '$lib/utils/state.svelte.js';

	import { CollectionManager, RecordClone } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import PipelineCard from './_partials/pipeline-card.svelte';
	import { runPipeline } from './_partials/utils';
	import { getAllPipelinesWorkflows } from './_partials/workflows.js';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Pipelines', right: navbarRight });

	const workflows = new PolledResource(getAllPipelinesWorkflows, {
		initialValue: () => data.workflows
	});
	$inspect(workflows.current);
</script>

<!-- Your Pipelines Section -->
<div class="mb-8">
	<T tag="h2" class="mb-4 text-lg font-semibold">{m.My()} {m.Pipelines()}</T>
	<CollectionManager
		collection="pipelines"
		queryOptions={{
			filter: `owner = '${organization.id}'`,
			sort: ['created', 'DESC'],
			expand: ['schedules_via_pipeline']
		}}
		hide={['pagination']}
	>
		{#snippet records({ records })}
			<div class="space-y-4">
				{#each records as pipeline, index (pipeline.id)}
					<PipelineCard bind:pipeline={records[index]} {organization} />
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
			filter: `owner != '${organization.id}'`,
			sort: ['created', 'DESC'],
			expand: ['owner']
		}}
		hide={['pagination', 'empty_state']}
	>
		{#snippet records({ records })}
			<div class="space-y-4">
				{#each records as pipeline (pipeline.id)}
					{@const ownerOrg = pipeline.expand?.owner}
					<DashboardCard
						record={pipeline}
						avatar={() =>
							ownerOrg
								? pb.files.getURL(ownerOrg, ownerOrg.logo)
								: pb.files.getURL(organization, organization.logo)}
						path={[organization.canonified_name, pipeline.canonified_name]}
						hideDelete={true}
						hidePublish={true}
					>
						{#snippet editAction()}
							<Button onclick={() => runPipeline(pipeline)}>
								<PlayIcon />{m.Run_now()}
							</Button>
							<RecordClone
								recordId={pipeline.id}
								size="md"
								collectionName="pipelines"
							/>
							<!-- <IconButton href="/my/pipelines/view-{pipeline.id}" icon={Eye} /> -->
						{/snippet}
					</DashboardCard>
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
