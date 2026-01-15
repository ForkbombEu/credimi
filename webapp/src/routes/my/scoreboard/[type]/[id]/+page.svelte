<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
import type { PageData } from './$types';

let { data }: { data: PageData } = $props();

function getAttributeValue(span: any, key: string): any {
return span.attributes.find((attr: any) => attr.key === key)?.value;
}

function formatDuration(startNano: number, endNano: number): string {
const durationMs = (endNano - startNano) / 1_000_000;
if (durationMs < 1000) {
return `${durationMs.toFixed(0)}ms`;
}
return `${(durationMs / 1000).toFixed(2)}s`;
}
</script>

<div class="container mx-auto p-6">
<div class="mb-6">
<a href="/my/scoreboard" class="text-blue-600 hover:text-blue-800">
← Back to Scoreboard
</a>
</div>

{#if !data.entry}
<div class="rounded-lg border border-yellow-300 bg-yellow-50 p-4 text-yellow-800">
<h2 class="font-semibold">Not Found</h2>
<p>The requested {data.type} with ID {data.id} was not found.</p>
</div>
{:else}
<h1 class="mb-6 text-3xl font-bold">{data.entry.name}</h1>

<!-- Summary Cards -->
<div class="mb-8 grid grid-cols-1 gap-6 md:grid-cols-4">
<div class="rounded-lg border bg-white p-6 shadow-sm">
<div class="text-sm text-gray-500">Total Runs</div>
<div class="mt-2 text-3xl font-bold">{data.entry.totalRuns}</div>
</div>
<div class="rounded-lg border bg-white p-6 shadow-sm">
<div class="text-sm text-gray-500">Successes</div>
<div class="mt-2 text-3xl font-bold text-green-600">{data.entry.successCount}</div>
</div>
<div class="rounded-lg border bg-white p-6 shadow-sm">
<div class="text-sm text-gray-500">Failures</div>
<div class="mt-2 text-3xl font-bold text-red-600">{data.entry.failureCount}</div>
</div>
<div class="rounded-lg border bg-white p-6 shadow-sm">
<div class="text-sm text-gray-500">Success Rate</div>
<div class="mt-2 text-3xl font-bold">{data.entry.successRate.toFixed(1)}%</div>
<div class="mt-2 h-2 overflow-hidden rounded-full bg-gray-200">
<div
class="h-full bg-green-500"
style="width: {data.entry.successRate}%"
></div>
</div>
</div>
</div>

<!-- Test Run History Chart Placeholder -->
<div class="mb-8 rounded-lg border bg-white p-6 shadow-sm">
<h2 class="mb-4 text-xl font-semibold">Test Run History</h2>
<div class="flex h-64 items-center justify-center bg-gray-50 text-gray-500">
<div class="text-center">
<p>Chart visualization placeholder</p>
<p class="text-sm">Success/Failure trends over time</p>
</div>
</div>
</div>

<!-- OpenTelemetry Spans -->
<div class="mb-8 rounded-lg border bg-white p-6 shadow-sm">
<h2 class="mb-4 text-xl font-semibold">Recent Test Runs (OpenTelemetry Data)</h2>
{#if data.spans.length > 0}
<div class="overflow-x-auto">
<table class="min-w-full divide-y divide-gray-200">
<thead class="bg-gray-50">
<tr>
<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
Trace ID
</th>
<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
Name
</th>
<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
Status
</th>
<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
Duration
</th>
<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
Success Rate
</th>
</tr>
</thead>
<tbody class="divide-y divide-gray-200 bg-white">
{#each data.spans as span}
<tr class="hover:bg-gray-50">
<td class="whitespace-nowrap px-6 py-4 text-sm font-mono text-gray-500">
{span.traceId.substring(0, 16)}...
</td>
<td class="whitespace-nowrap px-6 py-4 text-sm text-gray-900">
{span.name}
</td>
<td class="whitespace-nowrap px-6 py-4">
<span class="inline-flex rounded-full px-2 text-xs font-semibold leading-5 {span.status.code === 'OK' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
{span.status.code}
</span>
</td>
<td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
{formatDuration(span.startTimeUnixNano, span.endTimeUnixNano)}
</td>
<td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
{(() => {
const rate = getAttributeValue(span, 'test.success_rate');
return rate !== undefined && rate !== null ? `${rate}%` : 'N/A';
})()}
</td>
</tr>
{/each}
</tbody>
</table>
</div>
{:else}
<div class="text-center text-gray-500">No test run data available</div>
{/if}
</div>

<!-- Raw OpenTelemetry Data -->
<div class="rounded-lg border bg-white p-6 shadow-sm">
<details>
<summary class="cursor-pointer font-semibold">
Raw OpenTelemetry Data (Click to expand)
</summary>
<div class="mt-4">
<pre class="overflow-x-auto rounded bg-gray-100 p-4 text-sm">{JSON.stringify(
{ spans: data.spans },
null,
2
)}</pre>
</div>
</details>
</div>
{/if}
</div>
