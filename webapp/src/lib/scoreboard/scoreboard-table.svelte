<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ScoreboardEntry } from './types';

	interface Props {
		entries: ScoreboardEntry[];
		showActions?: boolean;
		emptyMessage?: string;
	}

	let { entries, showActions = false, emptyMessage = 'No data available' }: Props = $props();

	const columnCount = showActions ? 7 : 6;
</script>

<div class="overflow-x-auto rounded-lg border">
	<table class="min-w-full divide-y divide-gray-200">
		<thead class="bg-gray-50">
			<tr>
				<th
					class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
				>
					Name
				</th>
				<th
					class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
				>
					Total Runs
				</th>
				<th
					class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
				>
					Successes
				</th>
				<th
					class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
				>
					Failures
				</th>
				<th
					class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
				>
					Success Rate
				</th>
				<th
					class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
				>
					Last Run
				</th>
				{#if showActions}
					<th
						class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
					>
						Actions
					</th>
				{/if}
			</tr>
		</thead>
		<tbody class="divide-y divide-gray-200 bg-white">
			{#each entries as entry}
				<tr class="hover:bg-gray-50">
					<td class="whitespace-nowrap px-6 py-4">
						<div class="text-sm font-medium text-gray-900">{entry.name}</div>
					</td>
					<td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
						{entry.totalRuns}
					</td>
					<td class="whitespace-nowrap px-6 py-4 text-sm text-green-600">
						{entry.successCount}
					</td>
					<td class="whitespace-nowrap px-6 py-4 text-sm text-red-600">
						{entry.failureCount}
					</td>
					<td class="whitespace-nowrap px-6 py-4">
						<div class="flex items-center gap-2">
							<div class="h-2 w-24 overflow-hidden rounded-full bg-gray-200">
								<div class="h-full bg-green-500" style="width: {entry.successRate}%"></div>
							</div>
							<span class="text-sm text-gray-700">{entry.successRate.toFixed(1)}%</span>
						</div>
					</td>
					<td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
						{new Date(entry.lastRun).toLocaleDateString()}
					</td>
					{#if showActions}
						<td class="whitespace-nowrap px-6 py-4 text-sm">
							<a href="/my/scoreboard/{entry.type}/{entry.id}" class="text-blue-600 hover:text-blue-800">
								View Details
							</a>
						</td>
					{/if}
				</tr>
			{:else}
				<tr>
					<td colspan={columnCount} class="px-6 py-4 text-center text-gray-500">
						{emptyMessage}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
