<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { toUserTimezone } from '@/utils/toUserTimezone';
	import {
		workflowRun,
		fullEventHistory,
		currentEventHistory,
		toWorkflowExecution,
		WorkflowHistoryLayout,
		toEventHistory,
		type HistoryEvent,
		WorkflowStatus,
		calculateElapsedTime
		// pauseLiveUpdates
	} from '@forkbombeu/temporal-ui';

	//

	export let workflowResponse: Record<string, unknown>;
	export let eventHistory: HistoryEvent[];

	//

	let workflow = properToWorkflow(workflowResponse);
	$: workflow = properToWorkflow(workflowResponse);

	function properToWorkflow(workflowResponse: Record<string, unknown>) {
		// Note: Run this function ONLY IN THE BROWSER
		const w = toWorkflowExecution(workflowResponse);
		/* HACK */
		// canBeTerminated a property of workflow object is a getter that requires a svelte `store` to work
		// by removing it, we can avoid the store dependency and solve a svelte error about state not updating
		Object.defineProperty(w, 'canBeTerminated', {
			value: false
		});
		return w;
	}

	//

	// $pauseLiveUpdates = true;
	$: $workflowRun = { ...$workflowRun, workflow };
	$: $fullEventHistory = toEventHistory(eventHistory);
	$: $currentEventHistory = toEventHistory(eventHistory);

	//

	beforeNavigate(({ cancel, to }) => {
		const pathname = to?.url.pathname;
		if (pathname?.includes('undefined')) cancel();
	});
</script>

<div class="space-y-4 border-b-2 px-2 py-4 md:px-4 lg:px-8">
	{#if workflow.status}
		<WorkflowStatus status={workflow.status} />
	{/if}
	<table>
		<tbody>
			<tr>
				<td class="italic"> Start </td>
				<td class="pl-4">
					{toUserTimezone(workflow.startTime) ?? '-'}
				</td>
			</tr>
			<tr>
				<td class="italic"> End </td>
				<td class="pl-4">
					{toUserTimezone(workflow.endTime) ?? '-'}
				</td>
			</tr>
			<tr>
				<td class="italic"> Elapsed </td>
				<td class="pl-4">
					{calculateElapsedTime(workflow)}
				</td>
			</tr>
			<tr>
				<td class="italic"> Run ID </td>
				<td class="pl-4">
					{workflow.runId}
				</td>
			</tr>
		</tbody>
	</table>
</div>

<div class="temporal-ui-workflow space-y-4">
	<WorkflowHistoryLayout></WorkflowHistoryLayout>
</div>

<style lang="postcss">
	:global(div > table > tbody > div.text-right.hidden) {
		display: block;
	}

	:global(button.toggle-button[data-testid='download']) {
		display: none;
	}

	:global(button.toggle-button[data-testid='pause']) {
		display: none;
	}

	:global(.temporal-ui-workflow a) {
		@apply !no-underline hover:!cursor-not-allowed hover:!text-inherit;
	}
</style>
