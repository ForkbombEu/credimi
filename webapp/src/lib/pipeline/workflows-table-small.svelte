<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRightIcon, EllipsisVerticalIcon, ImageIcon, VideoIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { TemporalI18nProvider } from '$lib/temporal';

	import A from '@/components/ui-custom/a.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import { makeDropdownActions } from './actions';
	import WorkflowStatusTag from './workflow-status-tag.svelte';
	import * as PipelineWorkflows from './workflows';

	//

	type Props = {
		workflows: PipelineWorkflows.ExecutionSummary[];
	};

	let { workflows }: Props = $props();
</script>

<TemporalI18nProvider>
	<div class="overflow-hidden rounded-md border">
		<div class="overflow-x-auto">
			<table class="w-full text-xs">
				<thead class="bg-slate-100">
					<tr>
						<th>{m.Status()}</th>
						<th>{m.Runner()}</th>
						<th>{m.Results()}</th>
						<th>{m.Start_time()}</th>
						<th>{m.End_time()}</th>
						<th>{m.details()}</th>
						<th>{m.Actions()}</th>
					</tr>
				</thead>
				<tbody>
					{#each workflows as workflow (workflow.execution.runId)}
						{@const runnerNames = (workflow.runner_records ?? []).map((r) => r.name)}
						<tr>
							<td>
								<WorkflowStatusTag {workflow} size="sm" />
							</td>
							<td>
								{#if runnerNames.length > 0}
									{runnerNames.join(', ')}
								{:else}
									{@render na()}
								{/if}
							</td>

							<td>
								{#each workflow.results as result, index (index)}
									<div class="flex items-center gap-1">
										<IconButton
											size="mini"
											variant="ghost"
											icon={VideoIcon}
											href={result.video}
											target="_blank"
											class="text-primary hover:bg-secondary"
										/>
										<IconButton
											size="mini"
											variant="ghost"
											icon={ImageIcon}
											href={result.screenshot}
											target="_blank"
											class="text-primary hover:bg-secondary"
										/>
									</div>
								{:else}
									{@render na()}
								{/each}
							</td>
							<td class="text-muted-foreground">{workflow.startTime}</td>
							<td class="text-muted-foreground">
								{#if workflow.endTime !== ''}
									{workflow.endTime}
								{:else}
									{@render na()}
								{/if}
							</td>
							<td>
								<A
									href={resolve('/my/tests/runs/[workflow_id]/[run_id]', {
										workflow_id: workflow.execution.workflowId,
										run_id: workflow.execution.runId
									})}
								>
									{m.View()}
									<ArrowRightIcon class="inline-block size-3 -translate-y-px" />
								</A>
							</td>
							<td>
								<DropdownMenu
									triggerVariants={{ size: 'icon-sm', variant: 'ghost' }}
									items={makeDropdownActions(workflow)}
								>
									{#snippet triggerContent()}
										<EllipsisVerticalIcon />
									{/snippet}
								</DropdownMenu>
							</td>
						</tr>
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
		@apply text-left;
	}
</style>
