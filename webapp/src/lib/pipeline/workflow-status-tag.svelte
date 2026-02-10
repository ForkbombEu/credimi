<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { WorkflowStatusTag } from '$lib/workflows';

	import { m } from '@/i18n';

	import type { ExecutionSummary } from './workflows';

	//

	type Props = {
		workflow: ExecutionSummary;
		size?: ComponentProps<typeof WorkflowStatusTag>['size'];
	};

	let { workflow, size = 'md' }: Props = $props();
</script>

{#if workflow.queue}
	{@const { position, line_len } = workflow.queue}
	<p
		class={[
			'flex items-center gap-1 rounded-sm bg-yellow-200 px-1 py-0.5 text-sm font-medium whitespace-nowrap text-black',
			'w-fit origin-left',
			size === 'sm' && 'scale-75'
		]}
	>
		<span>
			{m.Queued()}
		</span>
		<span class="opacity-50">
			({position}/{line_len})
		</span>
	</p>
{:else}
	<WorkflowStatusTag status={workflow.status} failureReason={workflow.failure_reason} {size} />
{/if}
