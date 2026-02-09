<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { WorkflowStatusTag } from '$lib/workflows';

	import type { ExecutionSummary } from './workflows';

	//

	type Props = {
		workflow: ExecutionSummary;
	};

	let { workflow }: Props = $props();
</script>

{#if workflow.queue}
	<span class="text-muted-foreground opacity-50">Queued</span>
{:else}
	<WorkflowStatusTag status={workflow.status} failureReason={workflow.failure_reason} />
{/if}
