<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRightIcon, ImageIcon, VideoIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import WorkflowsTable from '$lib/workflows/workflows-table.svelte';

	import type { IconComponent } from '@/components/types';

	import A from '@/components/ui-custom/a.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
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
							{@render mediaPreview({
								image: result.screenshot,
								href: result.video,
								icon: VideoIcon
							})}
							{@render mediaPreview({
								image: result.screenshot,
								href: result.screenshot,
								icon: ImageIcon
							})}
						</div>
					{/each}
				</div>
			{:else}
				<span class="text-muted-foreground opacity-50">N/A</span>
			{/if}
		</Td>
	{/snippet}
</WorkflowsTable>

<!--  -->

{#snippet mediaPreview(props: { image: string; href: string; icon: IconComponent })}
	{@const { image, href, icon } = props}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a
		{href}
		target="_blank"
		class="relative size-10 shrink-0 overflow-hidden rounded-md border border-slate-300 hover:cursor-pointer hover:ring-2"
	>
		<img src={image} alt="Media" class="size-10 shrink-0" />
		<div class="absolute inset-0 flex items-center justify-center bg-black/30">
			<Icon src={icon} class="size-4  text-white" />
		</div>
	</a>
{/snippet}
