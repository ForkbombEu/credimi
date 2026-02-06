<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { TriangleIcon } from '@lucide/svelte';
	import clsx from 'clsx';
	import { slide } from 'svelte/transition';
	import { isWorkflowStatus } from '$lib/temporal';

	import A from '@/components/ui-custom/a.svelte';
	import Button from '@/components/ui-custom/button.svelte';

	import type { WorkflowExecutionSummary } from './queries.types';

	import WorkflowTreeBranch from './workflow-tree.svelte';

	//

	type Props = {
		workflow: WorkflowExecutionSummary;
		label?: string;
		root?: boolean;
	};

	let { workflow, label, root = false }: Props = $props();

	const { workflowId, runId } = $derived(workflow.execution);
	const status = $derived(() => {
		if (isWorkflowStatus(workflow.status)) return workflow.status;
		return 'Unspecified';
	});
	const hasChildren = $derived(workflow.children?.length && workflow.children.length > 0);

	let isExpanded = $state(true);
	// $effect(() => {
	// 	isExpanded = root;
	// });
</script>

<svelte:element this={root ? 'div' : 'li'}>
	<div class="flex items-center gap-1">
		<Button
			variant="ghost"
			size="icon"
			onclick={() => (isExpanded = !isExpanded)}
			class={[
				'size-5 shrink-0 hover:bg-secondary [&_svg]:size-2',
				{ invisible: !hasChildren }
			]}
		>
			<TriangleIcon
				stroke=""
				class={clsx('fill-primary stroke-none transition-all', {
					'rotate-180': isExpanded,
					'rotate-90': !isExpanded
				})}
			/>
		</Button>

		<A
			href={`/my/tests/runs/${workflowId}/${runId}`}
			class="flex items-center justify-start gap-3"
		>
			<span>
				{#if label}
					{label}
				{:else}
					{workflow.displayName}
				{/if}
			</span>
			{#if status !== null && !root}
				<div
					class="inline-block [&>div>span]:h-4 [&>div>span]:text-[8px] [&>div>span>.heart-beat]:hidden"
				>
					<WorkflowStatus {status} />
				</div>
			{/if}
		</A>
	</div>
	{#if hasChildren && isExpanded && workflow.children}
		<ul class="space-y-2 pt-2 pl-6" transition:slide>
			{#each workflow.children as child (child.execution.runId)}
				<WorkflowTreeBranch workflow={child} />
			{/each}
		</ul>
	{/if}
</svelte:element>
