<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

	import {
		ArrowRightIcon,
		ChevronDownIcon,
		ChevronUpIcon,
		EllipsisVerticalIcon
	} from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { TemporalI18nProvider } from '$lib/temporal';
	import {
		formatEndClock,
		splitExecutionTimes,
		type SplitExecutionTimes
	} from '$lib/workflows/format-execution-time';
	import { SvelteSet } from 'svelte/reactivity';
	import { fromStore } from 'svelte/store';

	import A from '@/components/ui-custom/a.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import { makeDropdownActions } from './actions';
	import { fromApiSummary } from './execution-artifacts';
	import ExecutionProgress from './execution-progress.svelte';
	import ExecutionArtifactsPreview from './results/execution-artifacts-preview.svelte';
	import WorkflowStatusTag from './workflow-status-tag.svelte';
	import { getExecutionDeviceNames, type ExecutionSummary } from './workflows';

	//

	type Props = {
		workflows: ExecutionSummary[];
	};

	let { workflows }: Props = $props();

	const user = fromStore(currentUser);
	const timezone = $derived(user.current?.Timezone);

	const expandedRunIds = new SvelteSet<string>();

	$effect(() => {
		const validIds = collectRunIds(workflows);
		for (const id of [...expandedRunIds]) {
			if (!validIds.has(id)) {
				expandedRunIds.delete(id);
			}
		}
	});

	function collectRunIds(
		items: Array<{ execution: { runId: string }; children?: WorkflowExecutionSummary[] }>,
		ids = new SvelteSet<string>()
	): SvelteSet<string> {
		for (const item of items) {
			ids.add(item.execution.runId);
			if (item.children?.length) {
				collectRunIds(item.children, ids);
			}
		}
		return ids;
	}

	function toggleChildren(runId: string) {
		if (expandedRunIds.has(runId)) {
			expandedRunIds.delete(runId);
		} else {
			expandedRunIds.add(runId);
		}
	}

	function statusPadding(depth: number): string {
		return `${depth * 1.5}rem`;
	}

	function parentTimeParts(workflow: ExecutionSummary): SplitExecutionTimes | undefined {
		if (workflow.queue) {
			if (!workflow.enqueuedAt) return undefined;
			return splitExecutionTimes(workflow.enqueuedAt, undefined, timezone);
		}
		return splitExecutionTimes(workflow.startTime, workflow.endTime, timezone);
	}
</script>

<TemporalI18nProvider>
	<div class="-mx-2 overflow-hidden">
		<div class="overflow-x-auto">
			<table class="w-full table-fixed text-xs">
				<thead class=" bg-slate-100">
					<tr>
						<th class="rounded-l-sm">{m.Status()}</th>
						<th>Device</th>
						<th>{m.Results()}</th>
						<th class="w-24 whitespace-nowrap">{m.Date()}</th>
						<th class="w-12 whitespace-nowrap">{m.start()}</th>
						<th class="w-20 whitespace-nowrap">{m.end()}</th>
						<th class="w-16 whitespace-nowrap">{m.Duration()}</th>
						<th class="w-24">{m.Children()}</th>
						<th class="w-14">{m.details()}</th>
						<th class="w-10 rounded-r-sm text-right!">{m.Actions()}</th>
					</tr>
				</thead>
				<tbody>
					{#each workflows as workflow (workflow.execution.runId)}
						{@const deviceNames = getExecutionDeviceNames(workflow)}
						{@const artifacts = fromApiSummary(workflow)}
						{@const children = (workflow.children ?? []) as WorkflowExecutionSummary[]}
						{@const count = children.length}
						{@const isExpanded =
							expandedRunIds.has(workflow.execution.runId) && count > 0}
						{@const parts = parentTimeParts(workflow)}
						<tr>
							<td>
								<WorkflowStatusTag
									status={workflow.status}
									queueData={workflow.queue}
									failureReason={workflow.failure_reason}
									size="sm"
								/>
								{#if workflow.status === 'Running'}
									<ExecutionProgress progress={workflow.progress} />
								{/if}
							</td>
							<td>
								{#if deviceNames.length > 0}
									{deviceNames.join(', ')}
								{:else}
									{@render na()}
								{/if}
							</td>

							<td>
								{#if artifacts}
									<ExecutionArtifactsPreview {artifacts} variant="compact" />
								{:else}
									{@render na()}
								{/if}
							</td>
							{@render timeCells(parts)}
							<td class="whitespace-nowrap text-muted-foreground">
								{#if workflow.duration}
									{workflow.duration}
								{:else}
									{@render na()}
								{/if}
							</td>
							<td class="max-w-24">
								{#if count > 0}
									<button
										type="button"
										class="flex max-w-full cursor-pointer items-center gap-1 text-left text-primary hover:underline"
										aria-expanded={isExpanded}
										onclick={() => toggleChildren(workflow.execution.runId)}
									>
										{#if isExpanded}
											<ChevronUpIcon class="size-3 shrink-0" />
										{:else}
											<ChevronDownIcon class="size-3 shrink-0" />
										{/if}
										<span class="min-w-0 truncate"
											>{m.count_children({ count })}</span
										>
									</button>
								{:else}
									<span class="text-muted-foreground opacity-50">—</span>
								{/if}
							</td>
							<td>
								{#if workflow.queue}
									{@render na()}
								{:else}
									<A
										href={resolve('/my/tests/runs/[workflow_id]/[run_id]', {
											workflow_id: workflow.execution.workflowId,
											run_id: workflow.execution.runId
										})}
										class="whitespace-nowrap"
									>
										{m.View()}
										<ArrowRightIcon
											class="inline-block size-3 -translate-y-px"
										/>
									</A>
								{/if}
							</td>
							<td class="text-right">
								<DropdownMenu
									items={makeDropdownActions(workflow)}
									triggerVariants={{ variant: 'ghost', size: 'icon-sm' }}
								>
									{#snippet trigger({ props })}
										<IconButton
											{...props}
											icon={EllipsisVerticalIcon}
											variant="ghost"
											size="xs"
										/>
									{/snippet}
								</DropdownMenu>
							</td>
						</tr>
						{#if isExpanded}
							{@render childRows(children, 1)}
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	</div>
</TemporalI18nProvider>

{#snippet timeCells(parts: SplitExecutionTimes | undefined)}
	<td class="whitespace-nowrap text-muted-foreground">
		{#if parts}
			{parts.date}
		{:else}
			{@render na()}
		{/if}
	</td>
	<td class="whitespace-nowrap text-muted-foreground">
		{#if parts}
			{parts.start}
		{:else}
			{@render na()}
		{/if}
	</td>
	<td class="whitespace-nowrap text-muted-foreground">
		{#if parts}
			{@const endClock = formatEndClock(parts)}
			{#if endClock}
				{endClock}
			{:else}
				{@render na()}
			{/if}
		{:else}
			{@render na()}
		{/if}
	</td>
{/snippet}

{#snippet childRows(children: WorkflowExecutionSummary[], depth: number)}
	{#each children as child (child.execution.runId)}
		{@const nested = child.children ?? []}
		{@const nestedCount = nested.length}
		{@const nestedExpanded = expandedRunIds.has(child.execution.runId) && nestedCount > 0}
		{@const childParts = splitExecutionTimes(child.startTime, child.endTime, timezone)}
		<tr class="bg-slate-50">
			<td style:padding-left={statusPadding(depth)}>
				<WorkflowStatusTag
					status={child.status}
					failureReason={child.failure_reason}
					size="sm"
				/>
			</td>
			<td colspan="2">
				<div class="flex min-w-0 items-baseline gap-1.5">
					<span class="shrink-0 font-normal">{child.type.name}</span>
					<span class="min-w-0 truncate text-muted-foreground">{child.displayName}</span>
				</div>
			</td>
			{@render timeCells(childParts)}
			<td class="whitespace-nowrap text-muted-foreground">
				{#if child.duration}
					{child.duration}
				{:else}
					{@render na()}
				{/if}
			</td>
			<td class="max-w-24">
				{#if nestedCount > 0}
					<button
						type="button"
						class="flex max-w-full cursor-pointer items-center gap-1 text-left text-primary hover:underline"
						aria-expanded={nestedExpanded}
						onclick={() => toggleChildren(child.execution.runId)}
					>
						{#if nestedExpanded}
							<ChevronUpIcon class="size-3 shrink-0" />
						{:else}
							<ChevronDownIcon class="size-3 shrink-0" />
						{/if}
						<span class="min-w-0 truncate"
							>{m.count_children({ count: nestedCount })}</span
						>
					</button>
				{:else}
					<span class="text-muted-foreground opacity-50">—</span>
				{/if}
			</td>
			<td>
				<A
					href={resolve('/my/tests/runs/[workflow_id]/[run_id]', {
						workflow_id: child.execution.workflowId,
						run_id: child.execution.runId
					})}
					class="whitespace-nowrap"
				>
					{m.View()}
					<ArrowRightIcon class="inline-block size-3 -translate-y-px" />
				</A>
			</td>
			<td></td>
		</tr>
		{#if nestedExpanded}
			{@render childRows(nested, depth + 1)}
		{/if}
	{/each}
{/snippet}

{#snippet na()}
	<span class="text-muted-foreground opacity-50">N/A</span>
{/snippet}

<style lang="postcss">
	@reference "tailwindcss";

	td,
	th {
		@apply px-2 py-0.5;
	}

	th {
		@apply text-left font-normal text-slate-500;
	}
</style>
