<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import { fromStore } from 'svelte/store';

	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import type { ExecutionSummary } from './workflows';

	import WorkflowsTableRow from './workflows-table-row.svelte';

	type Density = 'compact' | 'comfortable';

	type Props = {
		pipelineId: string;
		workflows: ExecutionSummary[];
		density?: Density;
		/** Called after a successful cancel from the row actions menu. */
		onCancel?: () => void;
	};

	let { pipelineId, workflows, density = 'comfortable', onCancel }: Props = $props();

	const isCompact = $derived(density === 'compact');
	const user = fromStore(currentUser);
	const timezone = $derived(user.current?.Timezone);

	const expandedRunIds = new SvelteSet<string>();
	const autoSeededRunIds = new SvelteSet<string>();

	$effect(() => {
		const validIds = new Set(workflows.map((item) => item.execution.runId));
		for (const id of [...expandedRunIds]) {
			if (!validIds.has(id)) {
				expandedRunIds.delete(id);
			}
		}
		for (const id of [...autoSeededRunIds]) {
			if (!validIds.has(id)) {
				autoSeededRunIds.delete(id);
			}
		}

		if (density === 'comfortable') {
			seedComfortableDefaults(workflows);
		}
	});

	/** Expands, once per run, the first run and every running one. */
	function seedComfortableDefaults(items: ExecutionSummary[]) {
		items.forEach((item, index) => {
			const id = item.execution.runId;
			const shouldExpand = !item.queue && (item.status === 'Running' || index === 0);
			if (shouldExpand && !autoSeededRunIds.has(id)) {
				autoSeededRunIds.add(id);
				expandedRunIds.add(id);
			}
		});
	}

	function toggleChildren(runId: string) {
		if (expandedRunIds.has(runId)) {
			expandedRunIds.delete(runId);
		} else {
			expandedRunIds.add(runId);
		}
	}
</script>

<div class="-mx-2 overflow-hidden">
	<div class="overflow-x-auto">
		<table
			class={['w-full', isCompact ? 'text-xs' : 'text-sm']}
			data-density={density}
			style:--td-px={isCompact ? '0.5rem' : '0.75rem'}
			style:--td-py={isCompact ? '0.125rem' : '0.5rem'}
		>
			<thead class="bg-slate-100">
				<tr>
					<th class="col-expand w-px rounded-l-sm whitespace-nowrap" aria-hidden="true"
					></th>
					<th>{m.Status()}</th>
					<th>{m.Devices()}</th>
					<th>{m.Results()}</th>
					<th>{m.Date()}</th>
					<th>{m.start()}</th>
					<th>{m.end()}</th>
					<th>{m.Duration()}</th>
					<th>{m.details()}</th>
					<th class="rounded-r-sm text-right!">{m.Actions()}</th>
				</tr>
			</thead>
			<tbody>
				{#each workflows as workflow, index (workflow.execution.runId)}
					<WorkflowsTableRow
						{pipelineId}
						{workflow}
						{density}
						{timezone}
						{expandedRunIds}
						onToggle={toggleChildren}
						{onCancel}
						showTopBorder={!isCompact && index > 0}
					/>
				{/each}
			</tbody>
		</table>
	</div>
</div>

<style lang="postcss">
	@reference "tailwindcss";

	th {
		@apply px-(--td-px) py-(--td-py) text-left font-normal text-slate-500;
	}

	th.col-expand {
		@apply px-0;
	}
</style>
