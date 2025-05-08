<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount } from 'svelte';
	import { QrCode } from '@/qr/index.js';
	import T from '@/components/ui-custom/t.svelte';

	let { data } = $props();

	onMount(() => {
		if (!data.workflowId) return;
		pb.send('/api/compliance/send-ewc-update-start', {
			method: 'POST',
			body: {
				workflow_id: data.workflowId
			}
		}).catch((err) => {
			console.error(err);
		});
	});

	function closeConnections() {
		if (!data.workflowId) return;
		pb.send('/api/compliance/send-ewc-update-stop', {
			method: 'POST',
			body: {
				workflow_id: data.workflowId
			}
		});
	}

	onDestroy(() => {
		closeConnections();
	});
</script>

<svelte:window on:beforeunload={closeConnections} />

<PageContent>
	<T tag="h1" class="mb-4">Wallet EWC test</T>

	<div>
		{#if data.qr}
			<QrCode src={data.qr} class="size-40 rounded-sm" />
		{:else}
			<T class="font-bold text-red-700">Error: QR code not found</T>
		{/if}
	</div>
</PageContent>
