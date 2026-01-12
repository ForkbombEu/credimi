<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import type { ScoreboardData } from './types';

	let scoreboardData: ScoreboardData | null = null;
	let loading = true;
	let error: string | null = null;
	let activeTab: 'wallets' | 'issuers' | 'verifiers' | 'pipelines' = 'wallets';

	async function fetchScoreboardData() {
		loading = true;
		error = null;
		try {
			const response = await fetch('/api/my/results');
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			scoreboardData = await response.json();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load scoreboard data';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchScoreboardData();
	});

	function getTabData() {
		if (!scoreboardData) return [];
		return scoreboardData.summary[activeTab] || [];
	}
</script>

<div class="container mx-auto p-6">
	<h1 class="mb-6 text-3xl font-bold">Scoreboard</h1>

	{#if loading}
		<div class="flex items-center justify-center p-12">
			<div class="text-lg">Loading scoreboard data...</div>
		</div>
	{:else if error}
		<div class="rounded-lg border border-red-300 bg-red-50 p-4 text-red-800">
			<h2 class="font-semibold">Error loading scoreboard</h2>
			<p>{error}</p>
		</div>
	{:else if scoreboardData}
		<!-- Tab Navigation -->
		<div class="mb-6 flex gap-2 border-b">
			<button
				class="px-4 py-2 {activeTab === 'wallets'
					? 'border-b-2 border-blue-600 font-semibold text-blue-600'
					: 'text-gray-600 hover:text-blue-600'}"
				on:click={() => (activeTab = 'wallets')}
			>
				Wallets ({scoreboardData.summary.wallets.length})
			</button>
			<button
				class="px-4 py-2 {activeTab === 'issuers'
					? 'border-b-2 border-blue-600 font-semibold text-blue-600'
					: 'text-gray-600 hover:text-blue-600'}"
				on:click={() => (activeTab = 'issuers')}
			>
				Issuers ({scoreboardData.summary.issuers.length})
			</button>
			<button
				class="px-4 py-2 {activeTab === 'verifiers'
					? 'border-b-2 border-blue-600 font-semibold text-blue-600'
					: 'text-gray-600 hover:text-blue-600'}"
				on:click={() => (activeTab = 'verifiers')}
			>
				Verifiers ({scoreboardData.summary.verifiers.length})
			</button>
			<button
				class="px-4 py-2 {activeTab === 'pipelines'
					? 'border-b-2 border-blue-600 font-semibold text-blue-600'
					: 'text-gray-600 hover:text-blue-600'}"
				on:click={() => (activeTab = 'pipelines')}
			>
				Pipelines ({scoreboardData.summary.pipelines.length})
			</button>
		</div>

		<!-- Data Table -->
		<div class="overflow-x-auto rounded-lg border">
			<table class="min-w-full divide-y divide-gray-200">
				<thead class="bg-gray-50">
					<tr>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Name
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Total Runs
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Successes
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Failures
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Success Rate
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Last Run
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
							Actions
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200 bg-white">
					{#each getTabData() as entry}
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
										<div
											class="h-full bg-green-500"
											style="width: {entry.successRate}%"
										></div>
									</div>
									<span class="text-sm text-gray-700">{entry.successRate.toFixed(1)}%</span>
								</div>
							</td>
							<td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
								{new Date(entry.lastRun).toLocaleDateString()}
							</td>
							<td class="whitespace-nowrap px-6 py-4 text-sm">
								<a
									href="/my/scoreboard/{entry.type}/{entry.id}"
									class="text-blue-600 hover:text-blue-800"
								>
									View Details
								</a>
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="7" class="px-6 py-4 text-center text-gray-500">
								No {activeTab} data available
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- OpenTelemetry Data Section -->
		<div class="mt-8">
			<details class="rounded-lg border">
				<summary class="cursor-pointer bg-gray-50 px-6 py-3 font-semibold">
					OpenTelemetry Data (Click to expand)
				</summary>
				<div class="p-6">
					<pre class="overflow-x-auto rounded bg-gray-100 p-4 text-sm">{JSON.stringify(
							scoreboardData.otelData,
							null,
							2
						)}</pre>
				</div>
			</details>
		</div>
	{/if}
</div>
