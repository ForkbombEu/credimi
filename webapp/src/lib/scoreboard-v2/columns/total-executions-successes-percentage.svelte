<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => {
			const total = row.total_runs ?? 0;
			const successes = row.total_successes ?? 0;
			const percent = row.success_rate;
			const manual = row.manually_executed_runs;
			const scheduled = row.scheduled_runs;
			return { total, successes, percent, manual, scheduled };
		},
		id: 'total_executions_successes_percentage',
		header: 'Success rate'
	});
</script>

<script lang="ts">
	import { ClockIcon, HandIcon } from '@lucide/svelte';

	let { value }: Column.Props<typeof column> = $props();
</script>

<div>
	<p class="text-xs font-bold">
		{value.successes}/{value.total} ({value.percent}%)
	</p>
	<p class="text-xs text-muted-foreground">
		{#if value.manual > 0}
			<span>
				{value.manual}
				<HandIcon class="inline-block size-3 -translate-px" />
			</span>
		{/if}
		{#if value.manual > 0 && value.scheduled > 0}
			<span> / </span>
		{/if}
		{#if value.scheduled > 0}
			<span>
				{value.scheduled}
				<ClockIcon class="inline-block size-3 -translate-px" />
			</span>
		{/if}
	</p>
</div>
