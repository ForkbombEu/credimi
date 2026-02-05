<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { CircleQuestionMarkIcon } from '@lucide/svelte';

	import Popover from '@/components/ui-custom/popover.svelte';

	import type { WorkflowExecutionSummary } from './queries.types';

	//

	type Props = {
		status: string | null;
		failureReason?: string;
		queue?: WorkflowExecutionSummary['queue'];
		size?: 'md' | 'sm';
	};

	let { status, failureReason, queue, size = 'md' }: Props = $props();

	//

	const readableStatus = $derived.by(() => {
		if (queue) return null;
		const s = toWorkflowStatusReadable(status);
		if (!s) throw new Error(`Invalid status: ${status}`);
		return s;
	});

	const queuePosition = $derived.by(() => {
		if (!queue) return null;
		const humanPosition = queue.position + 1;
		const lineLen = Math.max(queue.line_len, humanPosition);
		return { humanPosition, lineLen };
	});
</script>

<div class={['flex origin-left gap-1', size === 'sm' && 'scale-75']}>
	{#if queue}
		<div class="flex items-center gap-2">
			<span
				class={[
					'rounded-full border border-amber-200 bg-amber-100 px-2 py-0.5 text-[10px] uppercase tracking-wide text-amber-900',
					size === 'sm' && 'text-[9px]'
				]}
			>
				Queued
			</span>
			{#if queuePosition}
				<span class="text-xs text-muted-foreground">
					{queuePosition.humanPosition} of {queuePosition.lineLen}
				</span>
			{/if}
		</div>
	{:else}
		<WorkflowStatus status={readableStatus} />
		{#if failureReason}
			<Popover
				buttonVariants={{ variant: 'outline' }}
				containerClass="dark w-[400px]! p-3 text-xs"
				triggerClass="!h-6 !w-6 !p-0 text-xs underline"
			>
				{#snippet triggerContent()}
					<CircleQuestionMarkIcon class="size-4" />
				{/snippet}
				{#snippet content()}
					{failureReason}
				{/snippet}
			</Popover>
		{/if}
	{/if}
</div>
