<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRightIcon, InfoIcon, Pencil, RefreshCw } from '@lucide/svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import { Pipeline, Scoreboard } from '$lib';
	import { userOrganization } from '$lib/app-state';
	import StatusCircle from '$lib/components/status-circle.svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import PublishedSwitch from '$lib/layout/published-switch.svelte';
	import {
		invalidatePipelineListWorkflows,
		PIPELINE_LIST_WORKFLOWS_FAST_POLL_MS,
		PIPELINE_LIST_WORKFLOWS_IDLE_POLL_MS,
		PIPELINE_LIST_WORKFLOWS_LIMIT,
		PIPELINE_LIST_WORKFLOWS_MUTATION_BOOST_MS,
		pipelineListWorkflowsQueryKey,
		workflowsNeedFastPoll
	} from '$lib/pipeline/list-workflows-query';
	import { fromScoreboardRow } from '$lib/scoreboard/extras/from-scoreboard-row';
	import PipelineContentSummary from '$lib/scoreboard/extras/pipeline-content-summary.svelte';
	import PipelineExecutionStats from '$lib/scoreboard/extras/pipeline-execution-stats.svelte';
	import { getPath } from '$lib/utils';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Skeleton } from '@/components/ui/skeleton';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import { type EnrichedSchedule } from './types';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline', 'owner']>;
	};

	let { pipeline = $bindable() }: Props = $props();

	const queryClient = useQueryClient();

	/** Until this timestamp, poll fast even if the list has not yet shown Running/queue. */
	let fastPollUntil = $state(0);

	// Scheduling

	let schedule = $derived.by(() => {
		const s = pipeline.expand?.schedules_via_pipeline?.find(
			(schedule) => schedule.owner === userOrganization.current?.id
		);
		return s as EnrichedSchedule | undefined;
	});

	const scoreboard = createQuery(() => ({
		queryKey: ['pipeline-scoreboard', pipeline.id],
		queryFn: () => Scoreboard.Records.loadForPipeline(pipeline.id)
	}));

	const workflowsQuery = createQuery(() => {
		// Re-bind interval options when a mutation boost starts.
		void fastPollUntil;
		return {
			queryKey: pipelineListWorkflowsQueryKey(pipeline.id),
			queryFn: () =>
				Pipeline.Workflows.list(pipeline.id, {
					limit: PIPELINE_LIST_WORKFLOWS_LIMIT,
					page: 0
				}),
			refetchInterval: (query) => {
				if (Date.now() < fastPollUntil || workflowsNeedFastPoll(query.state.data)) {
					return PIPELINE_LIST_WORKFLOWS_FAST_POLL_MS;
				}
				return PIPELINE_LIST_WORKFLOWS_IDLE_POLL_MS;
			}
		};
	});

	function refreshWorkflows() {
		fastPollUntil = Date.now() + PIPELINE_LIST_WORKFLOWS_MUTATION_BOOST_MS;
		void invalidatePipelineListWorkflows(queryClient, pipeline.id);
	}

	// Variables for displaying UI elements

	const isPublic = $derived(pipeline.owner !== userOrganization.current?.id);
	const workflows = $derived(workflowsQuery.data);
	const workflowsLoading = $derived(workflowsQuery.isPending);
	const workflowsError = $derived(workflowsQuery.isError ? workflowsQuery.error : undefined);
	const isRunning = $derived(workflows?.some((workflow) => workflow.status === 'Running'));
	const showWorkflows = $derived(Boolean(workflows && workflows.length > 0));
	const showRunsSection = $derived(workflowsLoading || Boolean(workflowsError) || showWorkflows);

	const avatar = $derived.by(() => {
		const owner = pipeline.expand?.owner;
		if (!owner) return undefined;
		return pb.files.getURL(owner, owner.logo);
	});
</script>

<DashboardCard
	record={pipeline}
	{avatar}
	badge={isPublic ? m.Public() : undefined}
	hideActions={isPublic ? ['delete', 'edit', 'publish'] : undefined}
	{afterDescription}
	content={showRunsSection ? content : undefined}
	editAction={isPublic ? undefined : editAction}
	publishAction={isPublic ? undefined : publishAction}
	hideSeparator
>
	{#snippet nameRight()}
		{#if isRunning}
			<Badge
				variant="secondary"
				class="flex items-center gap-1.5 bg-green-100 text-green-800"
			>
				<StatusCircle size={12} />
				{m.Running()}
			</Badge>
		{/if}
	{/snippet}

	{#snippet actions()}
		<Pipeline.Device.RunNowButton {pipeline} onRun={refreshWorkflows} />
	{/snippet}

	{#snippet secondaryActions()}
		{#if !schedule}
			<SchedulePipelineForm {pipeline} />
		{:else}
			<ScheduleActions
				bind:schedule
				onCancel={() => {
					schedule = undefined;
				}}
			/>
		{/if}
	{/snippet}
</DashboardCard>

{#snippet publishAction()}
	<Tooltip>
		<PublishedSwitch record={pipeline} field="published" />
		{#snippet content()}
			<p>
				{pipeline.published ? m.pipeline_unpublish_tooltip() : m.pipeline_publish_tooltip()}
			</p>
		{/snippet}
	</Tooltip>
{/snippet}

{#snippet editAction()}
	<IconButton
		href={resolve('/my/pipelines/(group)/[...path]/edit', {
			path: getPath(pipeline, true)
		})}
		icon={Pencil}
		tooltip={pipeline.published ? m.pipeline_edit_disabled_while_published() : m.Edit()}
		disabled={pipeline.published}
	/>
{/snippet}

{#snippet afterDescription()}
	{#if scoreboard.data && Scoreboard.EntityDisplay.buildPipelineSummaryItems(scoreboard.data).length > 0}
		<div class="flex items-start justify-between gap-4 pt-1">
			<PipelineContentSummary results={scoreboard.data} />
		</div>
	{:else if scoreboard.isFetched}
		{@render emptyState()}
	{/if}
{/snippet}

{#snippet workflowsErrorBanner()}
	<div
		class="flex items-center justify-between gap-2 rounded-md border border-border bg-muted/60 px-3 py-2 text-xs text-muted-foreground"
	>
		<span>{m.Error()}</span>
		<IconButton
			icon={RefreshCw}
			variant="ghost"
			size="xs"
			tooltip={m.Error()}
			onclick={() => void workflowsQuery.refetch()}
		/>
	</div>
{/snippet}

{#snippet content()}
	<div class="space-y-3 pt-5">
		{#if workflowsLoading}
			<div class="space-y-2" aria-busy="true" aria-label={m.Loading()}>
				<Skeleton class="h-8 w-full rounded-md" />
				<Skeleton class="h-8 w-full rounded-md" />
				<Skeleton class="h-8 w-3/4 rounded-md" />
			</div>
		{:else}
			{#if workflowsError}
				{@render workflowsErrorBanner()}
			{/if}

			{#if showWorkflows && workflows}
				{@const executionStats = scoreboard.data
					? fromScoreboardRow(scoreboard.data)
					: undefined}
				<div class="space-y-3">
					<Pipeline.Workflows.SmallTable {workflows} onCancel={refreshWorkflows} />

					<div class="flex items-center justify-between gap-2">
						{#if executionStats}
							<PipelineExecutionStats stats={executionStats} layout="card-inline" />
						{:else}
							<div></div>
						{/if}

						<BlueButton
							compact
							href={resolve('/my/pipelines/[...pipeline_path]', {
								pipeline_path: getPath(pipeline, true)
							})}
						>
							{m.view_all()}
							<ArrowRightIcon />
						</BlueButton>
					</div>
				</div>
			{/if}
		{/if}
	</div>
{/snippet}

{#snippet emptyState()}
	<div class="flex items-center gap-2 text-xs text-muted-foreground opacity-50">
		<InfoIcon size={12} />
		<p>
			{m.Pipeline_summary_will_be_available_after_the_first_successful_run()}
		</p>
	</div>
{/snippet}
