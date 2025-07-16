<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { m } from '@/i18n/index.js';
	import Step from '../_partials/step.svelte';
	import QrLink from '../_partials/qr-link.svelte';
	import FeedbackForms from '../_partials/feedback-forms.svelte';
	import WorkflowLogs from '../_partials/workflow-logs.svelte';
	import { createOpenIdNetWorkflowManager } from '$lib/qrpages';
	import { onDestroy } from 'svelte';

	//

	let { data } = $props();
	const { qr, workflowId, namespace } = $derived(data);

	//

	let workflowManager: ReturnType<typeof createOpenIdNetWorkflowManager> | null = null;

	// Initialize the workflow manager when we have the required data
	$effect(() => {
		if (workflowId && namespace) {
			workflowManager?.destroy(); // Clean up previous instance
			workflowManager = createOpenIdNetWorkflowManager(workflowId, namespace);
		}
	});

	// Clean up on component destroy
	onDestroy(() => {
		workflowManager?.destroy();
	});

	// Setup beforeunload cleanup for cross-browser compatibility
	function handleBeforeUnload() {
		workflowManager?.destroy();
	}

	const workflowLogsProps = $derived.by(() => {
		return workflowManager?.getWorkflowLogsProps() || null;
	});
</script>

<svelte:window on:beforeunload={handleBeforeUnload} />

<PageContent>
	<T tag="h1" class="mb-4">Wallet OpenId test</T>
	<div class="space-y-4">
		{#if qr}
			<Step n="1" text="Scan this QR with the wallet app to start the check">
				<div
					class="bg-primary/10 ml-16 mt-4 flex flex-col items-center justify-center rounded-md p-2 sm:flex-row"
				>
					<QrLink {qr} />
				</div>
			</Step>
		{:else}
			<Alert variant="destructive">
				<T class="font-bold">{m.Error_check_failed()}</T>
				<T>
					{m.An_error_happened_during_the_check_please_read_the_logs_for_more_information()}
				</T>
			</Alert>
		{/if}

		{#if workflowId && namespace}
			<Step n="2" text="Follow the procedure on the wallet app">
				<div class="ml-16">
					{#if workflowLogsProps}
						<WorkflowLogs {...workflowLogsProps} />
					{:else}
						<p class="text-gray-500">Waiting for workflow logs...</p>
					{/if}
				</div>
			</Step>

			<Step n="3" text="Confirm the result">
				<FeedbackForms {workflowId} {namespace} />
			</Step>
		{/if}
	</div>
</PageContent>
