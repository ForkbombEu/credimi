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
			const percent = total === 0 ? 0 : (successes / total) * 100;
			return { total, successes, percent };
		},
		id: 'total_executions_successes_percentage',
		header: 'Total executions, successes, (% success)'
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

{JSON.stringify(value)}
