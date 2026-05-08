<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRightIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import MediaPreview from '$lib/components/media-preview.svelte';
	import WorkflowsTable from '$lib/workflows/workflows-table.svelte';

	import A from '@/components/ui-custom/a.svelte';
	import { m } from '@/i18n';

	import type { ExecutionSummary } from './workflows';

	import { makeDropdownActions } from './actions';
	import WorkflowStatusTag from './workflow-status-tag.svelte';

	//

	type Props = {
		workflows: ExecutionSummary[];
		hidePipelineColumn?: boolean;
	};

	let { workflows, hidePipelineColumn = false }: Props = $props();
</script>

<WorkflowsTable
	{workflows}
	hideColumns={['status', 'type']}
	actions={(w) => makeDropdownActions(w)}
	disableLink={(w) => w.queue !== undefined}
>
	{#snippet headerStart({ Th })}
		{#if !hidePipelineColumn}
			<Th>{m.Pipeline()}</Th>
		{/if}
	{/snippet}

	{#snippet header({ Th })}
		<Th>{m.Status()}</Th>
		<Th>{m.Runner()}</Th>
		<Th>{m.Results()}</Th>
	{/snippet}

	{#snippet rowStart({ workflow, Td, depth })}
		{#if !hidePipelineColumn}
			<Td>
				{#if depth === 0}
					<A
						href={resolve('/my/pipelines/[...pipeline_path]', {
							pipeline_path: workflow.pipeline_identifier ?? ''
						})}
						class="flex items-center gap-1"
					>
						<ArrowRightIcon size={12} />
						<span>
							{m.View()}
						</span>
					</A>
				{/if}
			</Td>
		{/if}
	{/snippet}

	{#snippet row({ workflow, Td })}
		{@const runnerNames = (workflow.runner_records ?? []).map((r) => r.name)}

		<Td>
			<WorkflowStatusTag
				status={workflow.status}
				queueData={workflow.queue}
				failureReason={workflow.failure_reason}
			/>
		</Td>

		<Td>
			{#if runnerNames.length > 0}
				{runnerNames.join(', ')}
			{:else}
				<span class="text-muted-foreground opacity-50">N/A</span>
			{/if}
		</Td>

		<Td>
			{#if workflow.results && workflow.results.length > 0}
				<div class="flex items-center gap-2">
					{#each workflow.results as result (result.video)}
						<div class="flex items-center gap-1">
							<MediaPreview image={result.video} href={result.video} icon="video" />
							<MediaPreview
								image={result.screenshot}
								href={result.screenshot}
								icon="image"
							/>
							<MediaPreview href={result.log} icon="file" />
						</div>
					{/each}
				</div>
			{:else}
				<span class="text-muted-foreground opacity-50">N/A</span>
			{/if}
		</Td>
	{/snippet}
</WorkflowsTable>
