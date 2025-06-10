<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount } from 'svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Step from '../_partials/step.svelte';
	import QrLink from '../_partials/qr-link.svelte';
	import { Alert } from '@/components/ui/alert';
	import { m } from '@/i18n';

	let { data } = $props();
	const { qr, workflowId, namespace } = $derived(data);

	onMount(() => {
		if (!workflowId) return;
		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId,
				namespace: namespace,
				signal: 'start-ewc-check-signal'
			}
		}).catch((err) => {
			console.error(err);
		});
	});

	function closeConnections() {
		if (!workflowId) return;
		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId,
				namespace: namespace,
				signal: 'stop-ewc-check-signal'
			}
		});
	}

	onDestroy(() => {
		closeConnections();
	});
</script>

<svelte:window on:beforeunload={closeConnections} />

<PageContent contentClass="space-y-4">
	<T tag="h1" class="mb-4">{m.Wallet_EWC_test()}</T>

	{#if qr}
		<Step n="1" text={m.Scan_this_QR_with_the_wallet_app_to_start_the_check()}>
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
			<!-- <div class="ml-16">
				<WorkflowLogs {workflowId} {namespace} />
			</div> -->
		</Step>
	{/if}
</PageContent>
