<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import clsx from 'clsx';
	import { TriangleIcon } from 'lucide-svelte';
	import { slide } from 'svelte/transition';

	import A from '@/components/ui-custom/a.svelte';
	import Button from '@/components/ui-custom/button.svelte';

	import type { WorkflowExecutionWithChildren } from './queries.types';

	import WorkflowTreeBranch from './workflow-tree.svelte';

	//

	type Props = {
		workflow: WorkflowExecutionWithChildren;
		label?: string;
		root?: boolean;
	};

	let { workflow, label, root = false }: Props = $props();

	const { workflowId, runId } = $derived(workflow.execution);
	const status = $derived(toWorkflowStatusReadable(workflow.status));
	const hasChildren = $derived(workflow.children?.length && workflow.children.length > 0);

	// eslint-disable-next-line svelte/prefer-writable-derived
	let isExpanded = $state(false);
	$effect(() => {
		isExpanded = root;
	});
</script>

<svelte:element this={root ? 'div' : 'li'}>
	<div class="flex items-center gap-1">
		<Button
			variant="ghost"
			size="icon"
			onclick={() => (isExpanded = !isExpanded)}
			class={[
				'hover:bg-secondary size-5 shrink-0 [&_svg]:size-2',
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
					class="inline-block [&>div>span>.heart-beat]:hidden [&>div>span]:h-4 [&>div>span]:text-[8px]"
				>
					<WorkflowStatus {status} />
				</div>
			{/if}
		</A>
	</div>
	{#if hasChildren && isExpanded && workflow.children}
		<ul class="space-y-2 pl-6 pt-2" transition:slide>
			{#each workflow.children as child (child.execution.runId)}
				<WorkflowTreeBranch workflow={child} />
			{/each}
		</ul>
	{/if}
</svelte:element>
