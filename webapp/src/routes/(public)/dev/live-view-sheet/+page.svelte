<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- THROWAWAY: delete this folder + the DEV mock in live-view.ts when done previewing. -->

<script lang="ts">
	import ExecutionDevices from '$lib/pipeline/execution-devices.svelte';
	import { LIVE_VIEW_PREVIEW_WORKFLOW_ID } from '$lib/pipeline/live-view';
	import type { ExecutionSummary } from '$lib/pipeline/workflows';

	const fakeExecution: ExecutionSummary = {
		execution: {
			workflowId: LIVE_VIEW_PREVIEW_WORKFLOW_ID,
			runId: 'preview-run'
		},
		type: { name: 'preview' },
		startTime: new Date().toISOString(),
		status: 'Running',
		displayName: 'Live view preview',
		devices: [
			{ device_id: 'preview/phone-a', name: 'preview-phone-a', live_view: true },
			{ device_id: 'preview/phone-b', name: 'preview-phone-b', live_view: true },
			{ device_id: 'preview/ios-sim', name: 'preview-ios-sim', live_view: false }
		]
	};
</script>

<main class="mx-auto max-w-lg space-y-4 p-8">
	<h1 class="text-lg font-semibold">Live view Sheet preview</h1>
	<p class="text-sm text-muted-foreground">
		Throwaway page (dev only). Click View live to open the Sheet with a same-origin fake stream.
	</p>
	<div class="rounded-md border bg-white p-4">
		<ExecutionDevices execution={fakeExecution} />
	</div>
</main>
