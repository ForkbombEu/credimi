<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { OTelDetails, type OTelSpan } from '$lib/scoreboard';

	import { m } from '@/i18n';

	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	function getAttributeValue(span: OTelSpan, key: string): unknown {
		return span.attributes.find((attr) => attr.key === key)?.value;
	}

	function formatDuration(startNano: number, endNano: number): string {
		const durationMs = (endNano - startNano) / 1_000_000;
		if (durationMs < 1000) {
			return `${durationMs.toFixed(0)}ms`;
		}
		return `${(durationMs / 1000).toFixed(2)}s`;
	}
</script>

<div class="page-container">
	<div class="back-link-container">
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<a href="/my/scoreboard" class="back-link">
			← {m.back_to_scoreboard()}
		</a>
	</div>

	{#if !data.entry}
		<div class="not-found-container">
			<h2 class="not-found-title">{m.not_found()}</h2>
			<p>{m.scoreboard_entity_not_found({ type: data.type, id: data.id })}</p>
		</div>
	{:else}
		<h1 class="page-title">{data.entry.name}</h1>

		<!-- Summary Cards -->
		<div class="summary-cards">
			<div class="summary-card">
				<div class="card-label">{m.scoreboard_total_runs()}</div>
				<div class="card-value">{data.entry.totalRuns}</div>
			</div>
			<div class="summary-card">
				<div class="card-label">{m.scoreboard_successes()}</div>
				<div class="card-value card-value-success">{data.entry.successCount}</div>
			</div>
			<div class="summary-card">
				<div class="card-label">{m.scoreboard_failures()}</div>
				<div class="card-value card-value-error">{data.entry.failureCount}</div>
			</div>
			<div class="summary-card">
				<div class="card-label">{m.scoreboard_success_rate()}</div>
				<div class="card-value">{data.entry.successRate.toFixed(1)}%</div>
				<div class="progress-bar-bg">
					<div class="progress-bar" style="width: {data.entry.successRate}%"></div>
				</div>
			</div>
		</div>

		<!-- Test Run History Chart Placeholder -->
		<div class="chart-container">
			<h2 class="section-title">{m.scoreboard_test_run_history()}</h2>
			<div class="chart-placeholder">
				<div class="chart-placeholder-text">
					<p>{m.chart_visualization_placeholder()}</p>
					<p class="chart-placeholder-subtitle">{m.success_failure_trends()}</p>
				</div>
			</div>
		</div>

		<!-- OpenTelemetry Spans -->
		<div class="spans-container">
			<h2 class="section-title">{m.scoreboard_recent_test_runs()}</h2>
			{#if data.spans.length > 0}
				<div class="table-wrapper">
					<table class="spans-table">
						<thead class="table-head">
							<tr>
								<th class="table-header">{m.trace_id()}</th>
								<th class="table-header">{m.name()}</th>
								<th class="table-header">{m.status()}</th>
								<th class="table-header">{m.duration()}</th>
								<th class="table-header">{m.scoreboard_success_rate()}</th>
							</tr>
						</thead>
						<tbody class="table-body">
							{#each data.spans as span (span.traceId)}
								<tr class="table-row">
									<td class="cell-mono table-cell">
										{span.traceId.substring(0, 16)}...
									</td>
									<td class="table-cell">
										{span.name}
									</td>
									<td class="table-cell">
										<span
											class="status-badge"
											class:status-ok={span.status.code === 'OK'}
											class:status-error={span.status.code !== 'OK'}
										>
											{span.status.code}
										</span>
									</td>
									<td class="cell-secondary table-cell">
										{formatDuration(
											span.startTimeUnixNano,
											span.endTimeUnixNano
										)}
									</td>
									<td class="cell-secondary table-cell">
										{(() => {
											const rate = getAttributeValue(
												span,
												'test.success_rate'
											);
											return rate !== undefined && rate !== null
												? `${rate}%`
												: m.not_available();
										})()}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="empty-message">{m.no_test_run_data_available()}</div>
			{/if}
		</div>

		<!-- Raw OpenTelemetry Data -->
		<OTelDetails title={m.raw_opentelemetry_data()} content={{ spans: data.spans }} />
	{/if}
</div>

<style lang="postcss">
	.page-container {
		@apply container mx-auto p-6;
	}

	.back-link-container {
		@apply mb-6;
	}

	.back-link {
		@apply text-blue-600 hover:text-blue-800;
	}

	.not-found-container {
		@apply rounded-lg border border-yellow-300 bg-yellow-50 p-4 text-yellow-800;
	}

	.not-found-title {
		@apply font-semibold;
	}

	.page-title {
		@apply mb-6 text-3xl font-bold;
	}

	.summary-cards {
		@apply mb-8 grid grid-cols-1 gap-6 md:grid-cols-4;
	}

	.summary-card {
		@apply rounded-lg border bg-white p-6 shadow-sm;
	}

	.card-label {
		@apply text-sm text-gray-500;
	}

	.card-value {
		@apply mt-2 text-3xl font-bold;
	}

	.card-value-success {
		@apply text-green-600;
	}

	.card-value-error {
		@apply text-red-600;
	}

	.progress-bar-bg {
		@apply mt-2 h-2 overflow-hidden rounded-full bg-gray-200;
	}

	.progress-bar {
		@apply h-full bg-green-500;
	}

	.chart-container {
		@apply mb-8 rounded-lg border bg-white p-6 shadow-sm;
	}

	.section-title {
		@apply mb-4 text-xl font-semibold;
	}

	.chart-placeholder {
		@apply flex h-64 items-center justify-center bg-gray-50 text-gray-500;
	}

	.chart-placeholder-text {
		@apply text-center;
	}

	.chart-placeholder-subtitle {
		@apply text-sm;
	}

	.spans-container {
		@apply mb-8 rounded-lg border bg-white p-6 shadow-sm;
	}

	.table-wrapper {
		@apply overflow-x-auto;
	}

	.spans-table {
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
		@apply whitespace-nowrap px-6 py-4 text-sm text-gray-900;
	}

	.cell-mono {
		@apply font-mono text-gray-500;
	}

	.cell-secondary {
		@apply text-gray-500;
	}

	.status-badge {
		@apply inline-flex rounded-full px-2 text-xs font-semibold leading-5;
	}

	.status-ok {
		@apply bg-green-100 text-green-800;
	}

	.status-error {
		@apply bg-red-100 text-red-800;
	}

	.empty-message {
		@apply text-center text-gray-500;
	}
</style>
