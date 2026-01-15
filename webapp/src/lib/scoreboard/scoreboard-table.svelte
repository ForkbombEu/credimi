<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';
	import type { ScoreboardEntry } from './types';

	interface Props {
		entries: ScoreboardEntry[];
		showActions?: boolean;
		emptyMessage?: string;
	}

	let { entries, showActions = false, emptyMessage = m.no_data_available() }: Props = $props();

	const columnCount = showActions ? 7 : 6;
</script>

<div class="table-wrapper">
	<table class="table">
		<thead class="table-head">
			<tr>
				<th class="table-header">
					{m.name()}
				</th>
				<th class="table-header">
					{m.scoreboard_total_runs()}
				</th>
				<th class="table-header">
					{m.scoreboard_successes()}
				</th>
				<th class="table-header">
					{m.scoreboard_failures()}
				</th>
				<th class="table-header">
					{m.scoreboard_success_rate()}
				</th>
				<th class="table-header">
					{m.scoreboard_last_run()}
				</th>
				{#if showActions}
					<th class="table-header">
						{m.actions()}
					</th>
				{/if}
			</tr>
		</thead>
		<tbody class="table-body">
			{#each entries as entry}
				<tr class="table-row">
					<td class="table-cell">
						<div class="cell-name">{entry.name}</div>
					</td>
					<td class="table-cell cell-secondary">
						{entry.totalRuns}
					</td>
					<td class="table-cell cell-success">
						{entry.successCount}
					</td>
					<td class="table-cell cell-error">
						{entry.failureCount}
					</td>
					<td class="table-cell">
						<div class="progress-container">
							<div class="progress-bar-bg">
								<div class="progress-bar" style="width: {entry.successRate}%"></div>
							</div>
							<span class="progress-text">{entry.successRate.toFixed(1)}%</span>
						</div>
					</td>
					<td class="table-cell cell-secondary">
						{new Date(entry.lastRun).toLocaleDateString()}
					</td>
					{#if showActions}
						<td class="table-cell">
							<a href="/my/scoreboard/{entry.type}/{entry.id}" class="action-link">
								{m.view_details()}
							</a>
						</td>
					{/if}
				</tr>
			{:else}
				<tr>
					<td colspan={columnCount} class="empty-cell">
						{emptyMessage}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style lang="postcss">
	.table-wrapper {
		@apply overflow-x-auto rounded-lg border;
	}

	.table {
		@apply min-w-full divide-y divide-gray-200;
	}

	.table-head {
		@apply bg-gray-50;
	}

	.table-header {
		@apply px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500;
	}

	.table-body {
		@apply divide-y divide-gray-200 bg-white;
	}

	.table-row {
		@apply hover:bg-gray-50;
	}

	.table-cell {
		@apply whitespace-nowrap px-6 py-4;
	}

	.cell-name {
		@apply text-sm font-medium text-gray-900;
	}

	.cell-secondary {
		@apply text-sm text-gray-500;
	}

	.cell-success {
		@apply text-sm text-green-600;
	}

	.cell-error {
		@apply text-sm text-red-600;
	}

	.progress-container {
		@apply flex items-center gap-2;
	}

	.progress-bar-bg {
		@apply h-2 w-24 overflow-hidden rounded-full bg-gray-200;
	}

	.progress-bar {
		@apply h-full bg-green-500;
	}

	.progress-text {
		@apply text-sm text-gray-700;
	}

	.action-link {
		@apply text-blue-600 hover:text-blue-800;
	}

	.empty-cell {
		@apply px-6 py-4 text-center text-gray-500;
	}
</style>
