<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { WorkflowStatusTag } from '$lib/workflows';

	import { m } from '@/i18n';

	import type { ExecutionSummary, Status } from './workflows';

	//

	type Props = Omit<ComponentProps<typeof WorkflowStatusTag>, 'status'> & {
		status: Status;
		queueData?: ExecutionSummary['queue'];
	};

	let { status, size = 'md', queueData, ...props }: Props = $props();
</script>

{#if status === 'Queued'}
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
		{#if queueData}
			<span class="opacity-50">
				({queueData.position}/{queueData.line_len})
			</span>
		{/if}
	</p>
{:else}
	<WorkflowStatusTag {status} {size} {...props} />
{/if}
