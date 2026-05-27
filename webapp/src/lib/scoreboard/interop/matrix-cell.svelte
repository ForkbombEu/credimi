<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	import type { InteropMatrixCell } from './types';
	import { interopStatusStyles } from './status';

	type Props = {
		cell?: InteropMatrixCell;
	};

	let { cell }: Props = $props();

	const styles = $derived(cell ? interopStatusStyles(cell.status) : null);
	const displayPercent = $derived(cell ? Math.round(cell.success_rate) : 0);
	const pipelineLabel = $derived(
		cell
			? cell.pipeline_count === 1
				? m.interop_matrix_pipeline_count_one({ count: cell.pipeline_count })
				: m.interop_matrix_pipeline_count_other({ count: cell.pipeline_count })
			: ''
	);
</script>

{#if cell && styles}
	<div
		class="flex min-h-24 flex-col items-center justify-center gap-1 rounded-md p-3 text-center {styles.bg}"
	>
		<p class="text-2xl font-bold {styles.text}">{displayPercent}%</p>
		<p class="text-sm font-medium {styles.text}">
			{cell.total_successes}/{cell.total_runs}
		</p>
		<p class="mt-1 text-xs text-muted-foreground">{pipelineLabel}</p>
	</div>
{:else}
	<div
		class="flex min-h-24 items-center justify-center rounded-md bg-muted/40 p-3 text-center text-sm text-muted-foreground"
	>
		{m.interop_matrix_not_tested()}
	</div>
{/if}
