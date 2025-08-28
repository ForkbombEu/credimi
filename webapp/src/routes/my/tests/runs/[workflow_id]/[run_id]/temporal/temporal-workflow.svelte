<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import {
		workflowRun,
		fullEventHistory,
		currentEventHistory,
		WorkflowHistoryLayout,
		toEventHistory,
		type HistoryEvent
	} from '@forkbombeu/temporal-ui';
	import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';
	import _ from 'lodash';
	import { fly } from 'svelte/transition';

	//

	type Props = {
		execution: WorkflowExecution;
		eventHistory: HistoryEvent[];
		onLoad?: () => void;
	};

	let { execution, eventHistory, onLoad }: Props = $props();

	/* Updating temporal stores */

	$effect(() => {
		const wf = $state.snapshot(execution);
		if (!_.isEqual(wf, $workflowRun.workflow)) {
			workflowRun.update((value) => ({
				...value,
				workflow: wf
			}));
		}
	});

	$effect(() => {
		const history = toEventHistory($state.snapshot(eventHistory));
		if (_.isEqual(history, $currentEventHistory) || _.isEqual(history, $fullEventHistory)) {
			return;
		}
		fullEventHistory.set(history);
		currentEventHistory.set(history);
	});

	/* Preventing navigation to undefined pages */

	beforeNavigate(({ cancel, to }) => {
		const pathname = to?.url.pathname;
		if (pathname?.includes('undefined')) cancel();
	});

	/* */

	$effect(() => {
		onLoad?.();
	});
</script>

{#if $workflowRun.workflow}
	<div class="temporal-ui-workflow space-y-4" in:fly={{ duration: 1000 }}>
		<WorkflowHistoryLayout></WorkflowHistoryLayout>
	</div>
{/if}

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
