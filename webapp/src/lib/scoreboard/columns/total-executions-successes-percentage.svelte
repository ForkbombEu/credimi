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
			const ci = (row as typeof row & { CI_runs?: number }).CI_runs ?? 0;
			return { total, successes, percent, manual, scheduled, ci };
		},
		id: 'total_executions_successes_percentage',
		header: m.scoreboard_success_rate(),
		sortField: 'success_rate'
	});
</script>

<script lang="ts">
	import PipelineExecutionStats from '../extras/pipeline-execution-stats.svelte';

	let { value }: Column.Props<typeof column> = $props();
</script>

<PipelineExecutionStats stats={value} layout="inline" />
