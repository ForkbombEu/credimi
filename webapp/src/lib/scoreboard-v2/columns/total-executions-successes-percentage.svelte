<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => {
			const total = row.total_runs ?? 0;
			const successes = row.total_successes ?? 0;
			const percent = row.success_rate;
			const manual = row.manually_executed_runs;
			const scheduled = row.scheduled_runs;
			const ci = 0; // TODO: add CI executions
			return { total, successes, percent, manual, scheduled, ci };
		},
		id: 'total_executions_successes_percentage',
		header: m.scoreboard_success_rate()
	});
</script>

<script lang="ts">
	import { ClockIcon, CogIcon, HandIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import Tooltip from '@/components/ui-custom/tooltip.svelte';

	//

	let { value }: Column.Props<typeof column> = $props();

	type ExecutionModeCount = {
		icon: IconComponent;
		count: number;
		label: string;
	};

	const executionTypes: ExecutionModeCount[] = $derived([
		{ icon: HandIcon, count: value.manual, label: m.Executed_manually() },
		{
			icon: ClockIcon,
			count: value.scheduled,
			label: m.Executed_via_scheduling()
		},
		{ icon: CogIcon, count: value.ci, label: m.Executed_via_ci() }
	]);
</script>

<div class="pr-3">
	<p class={['text-sm font-bold', { 'text-emerald-600': value.percent >= 70 }]}>
		{value.successes}/{value.total} ({value.percent}%)
	</p>
	<p class="text-xs text-muted-foreground opacity-80">
		{#each executionTypes as executionType, index (executionType.label)}
			<Tooltip>
				<span>
					{executionType.count}
					<executionType.icon class="-ml-0.5 inline-block size-3 -translate-y-px" />
				</span>

				{#snippet content()}
					<p>
						<executionType.icon class="inline-block size-3 -translate-y-px" />
						{executionType.label}
					</p>
				{/snippet}
			</Tooltip>
			{#if index < executionTypes.length - 1}
				<span class="pr-1 pl-0.5">/</span>
			{/if}
		{/each}
	</p>
</div>
