<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

	import { ArrowRightIcon, Pencil } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { Pipeline, Scoreboard } from '$lib';
	import { userOrganization } from '$lib/app-state';
	import StatusCircle from '$lib/components/status-circle.svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import PublishedSwitch from '$lib/layout/published-switch.svelte';
	import { fromScoreboardRow } from '$lib/scoreboard/extras/from-scoreboard-row';
	import PipelineContentSummary from '$lib/scoreboard/extras/pipeline-content-summary.svelte';
	import PipelineExecutionStats from '$lib/scoreboard/extras/pipeline-execution-stats.svelte';
	import { getPath } from '$lib/utils';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import { type EnrichedSchedule } from './types';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline', 'owner']>;
		workflows?: WorkflowExecutionSummary[];
		onRun?: () => void;
	};

	let { pipeline = $bindable(), workflows, onRun }: Props = $props();

	// Scheduling

	let schedule = $derived.by(() => {
		const s = pipeline.expand?.schedules_via_pipeline?.find(
			(schedule) => schedule.owner === userOrganization.current?.id
		);
		return s as EnrichedSchedule | undefined;
	});

	const scoreboardPromise = $derived.by(() =>
		Scoreboard.Records.loadForPipeline(pipeline.id).catch((error) => {
			console.error(error);
			return undefined;
		})
	);

	// Variables for displaying UI elements

	const isPublic = $derived(pipeline.owner !== userOrganization.current?.id);
	const isRunning = $derived(workflows?.some((workflow) => workflow.status === 'Running'));
	const showContent = $derived(workflows && workflows.length > 0);

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
	content={showContent ? content : undefined}
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
		<Pipeline.Runner.RunNowButton {pipeline} {onRun} />

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
	{#await scoreboardPromise}
		{@render emptyState()}
	{:then results}
		{#if results && Scoreboard.EntityDisplay.buildPipelineSummaryItems(results).length > 0}
			<div class="flex items-start justify-between gap-4 pt-1">
				<PipelineContentSummary {results} />
			</div>
		{:else}
			{@render emptyState()}
		{/if}
	{:catch}
		{@render emptyState()}
	{/await}
{/snippet}

{#snippet content()}
	<div class="space-y-3">
		{#if workflows && workflows.length > 0}
			<div class="space-y-3 pt-5">
				<Pipeline.Workflows.SmallTable {workflows} />

				<div class="flex items-center justify-between gap-2">
					{#await scoreboardPromise}
						<!-- pending -->
					{:then results}
						{@const executionStats = results ? fromScoreboardRow(results) : undefined}
						{#if executionStats}
							<PipelineExecutionStats stats={executionStats} layout="card-inline" />
						{/if}
					{/await}

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
	</div>
{/snippet}

{#snippet emptyState()}
	<div
		class="flex h-8 w-fit items-center justify-start rounded-md bg-muted p-2 text-xs text-muted-foreground"
	>
		{m.Pipeline_summary_will_be_available_after_the_first_successful_run()}
	</div>
{/snippet}
