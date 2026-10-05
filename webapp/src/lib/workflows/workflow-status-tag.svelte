<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CircleQuestionMarkIcon } from '@lucide/svelte';

	import FailureText from '@/components/ui-custom/failure-text.svelte';
	import Popover from '@/components/ui-custom/popover.svelte';

	import type { WorkflowStatus } from './types';

	import WorkflowStatusBadge from './workflow-status.svelte';

	type Props = {
		status: WorkflowStatus;
		failureReason?: string;
		size?: 'md' | 'sm';
	};

	let { status, failureReason, size = 'md' }: Props = $props();
</script>

<div class={['flex origin-left gap-1', size === 'sm' && 'scale-75']}>
	<WorkflowStatusBadge {status} />
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
				<FailureText>{failureReason}</FailureText>
			{/snippet}
		</Popover>
	{/if}
</div>
