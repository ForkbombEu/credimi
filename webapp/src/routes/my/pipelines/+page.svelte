<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { getPath, runWithLoading } from '$lib/utils';
	import { Pencil, PlayIcon, Plus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import { CollectionManager, RecordClone } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { goto, m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Pipelines', right: navbarRight });

	//

	async function runPipeline(pipeline: PipelinesResponse) {
		const result = await runWithLoading({
			fn: async () => {
				return await pb.send('/api/pipeline/start', {
					method: 'POST',
					body: {
						yaml: pipeline.yaml,
						pipeline_identifier: getPath(pipeline)
					}
				});
			},
			showSuccessToast: false
		});

		if (result?.result) {
			const { workflowId, workflowRunId } = result.result;
			const workflowUrl =
				workflowId && workflowRunId
					? `/my/tests/runs/${workflowId}/${workflowRunId}`
					: undefined;

			toast.success(m.Pipeline_started_successfully(), {
				description: m.View_workflow_details(),
				duration: 10000,
				...(workflowUrl && {
					action: {
						label: m.View(),
						onClick: () => goto(workflowUrl)
					}
				})
			});
		}
	}
</script>

<!-- Your Pipelines Section -->
<div class="mb-8">
	<T tag="h2" class="mb-4 text-lg font-semibold">{m.My()} {m.Pipelines()}</T>
	<CollectionManager
		collection="pipelines"
		queryOptions={{
			filter: `owner = '${organization.id}'`,
			sort: ['created', 'DESC']
		}}
		hide={['pagination']}
	>
		{#snippet records({ records })}
			<div class="space-y-4">
				{#each records as pipeline (pipeline.id)}
					<DashboardCard
						record={pipeline}
						avatar={() => pb.files.getURL(organization, organization.logo)}
						path={[organization.canonified_name, pipeline.canonified_name]}
						badge={m.Yours()}
					>
						{#snippet editAction()}
							<Button onclick={() => runPipeline(pipeline)}>
								<PlayIcon />{m.Run_now()}
							</Button>
							<!-- <Button
								href="/my/pipelines/settings-{pipeline.id}"
								variant="outline"
								size="icon"
							>
								<CogIcon />
							</Button> -->
							<RecordClone record={pipeline} size="md" collectionName="pipelines" />
							<IconButton href="/my/pipelines/edit-{pipeline.id}" icon={Pencil} />
						{/snippet}
					</DashboardCard>
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
							<RecordClone record={pipeline} size="md" collectionName="pipelines" />
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
