<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { CircleQuestionMarkIcon } from '@lucide/svelte';

	import Popover from '@/components/ui-custom/popover.svelte';

	//

	type Props = {
		status: string | null;
		failureReason?: string;
		size?: 'md' | 'sm';
	};

	let { status, failureReason, size = 'md' }: Props = $props();

	//

	const readableStatus = $derived.by(() => {
		const s = toWorkflowStatusReadable(status);
		if (!s) throw new Error(`Invalid status: ${status}`);
		return s;
	});
</script>

<div class={['flex origin-left gap-1', size === 'sm' && 'scale-75']}>
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
</div>
