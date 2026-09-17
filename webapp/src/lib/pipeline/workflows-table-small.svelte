<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

	import { ArrowRightIcon, EllipsisVerticalIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { TemporalI18nProvider } from '$lib/temporal';

	import A from '@/components/ui-custom/a.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

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

	const PARENT_COLUMN_COUNT = 9;

	let expandedRunId = $state<string | null>(null);

	$effect(() => {
		if (
			expandedRunId &&
			!workflows.some((workflow) => workflow.execution.runId === expandedRunId)
		) {
			expandedRunId = null;
		}
	});

	function toggleChildren(runId: string) {
		expandedRunId = expandedRunId === runId ? null : runId;
	}
</script>

<TemporalI18nProvider>
	<div class="-mx-2 overflow-hidden">
		<div class="overflow-x-auto">
			<table class="w-full text-xs">
				<thead class=" bg-slate-100">
					<tr>
						<th class="rounded-l-sm">{m.Status()}</th>
						<th>Device</th>
						<th>{m.Results()}</th>
						<th>{m.Start_time()}</th>
						<th>{m.End_time()}</th>
						<th>{m.Duration()}</th>
						<th>{m.Children()}</th>
						<th>{m.details()}</th>
						<th class="rounded-r-sm text-right!">{m.Actions()}</th>
					</tr>
				</thead>
				<tbody>
					{#each workflows as workflow (workflow.execution.runId)}
						{@const deviceNames = getExecutionDeviceNames(workflow)}
						{@const artifacts = fromApiSummary(workflow)}
						{@const children = (workflow.children ?? []) as WorkflowExecutionSummary[]}
						{@const count = children.length}
						{@const isExpanded =
							expandedRunId === workflow.execution.runId && count > 0}
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
							<td class="text-muted-foreground">
								{#if workflow.queue}
									{@render na()}
								{:else}
									{workflow.startTime}
								{/if}
							</td>
							<td class="text-muted-foreground">
								{#if workflow.endTime !== ''}
									{workflow.endTime}
								{:else}
									{@render na()}
								{/if}
							</td>
							<td class="text-muted-foreground">
								{#if workflow.duration}
									{workflow.duration}
								{:else}
									{@render na()}
								{/if}
							</td>
							<td>
								{#if count > 0}
									<button
										type="button"
										class="text-primary hover:underline"
										aria-expanded={isExpanded}
										onclick={() => toggleChildren(workflow.execution.runId)}
									>
										{m.count_children({ count })}
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
							<tr class="bg-slate-50">
								<td colspan={PARENT_COLUMN_COUNT} class="px-2 py-2">
									<table class="w-full text-xs">
										<thead>
											<tr>
												<th class="rounded-l-sm">{m.Status()}</th>
												<th>{m.Type()}</th>
												<th>{m.Duration()}</th>
												<th class="rounded-r-sm">{m.details()}</th>
											</tr>
										</thead>
										<tbody>
											{#each children as child (child.execution.runId)}
												<tr>
													<td>
														<WorkflowStatusTag
															status={child.status}
															failureReason={child.failure_reason}
															size="sm"
														/>
													</td>
													<td>
														<div class="flex min-w-0 flex-col gap-0.5">
															<span class="truncate font-medium">
																{child.type.name}
															</span>
															<span
																class="truncate text-muted-foreground"
															>
																{child.displayName}
															</span>
														</div>
													</td>
													<td class="text-muted-foreground">
														{#if child.duration}
															{child.duration}
														{:else}
															{@render na()}
														{/if}
													</td>
													<td>
														<A
															href={resolve(
																'/my/tests/runs/[workflow_id]/[run_id]',
																{
																	workflow_id:
																		child.execution.workflowId,
																	run_id: child.execution.runId
																}
															)}
														>
															{m.View()}
															<ArrowRightIcon
																class="inline-block size-3 -translate-y-px"
															/>
														</A>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	</div>
</TemporalI18nProvider>

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
