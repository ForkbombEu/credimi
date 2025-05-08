<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount } from 'svelte';
	import ParamsChecker from '../_partials/params-checker.svelte';
	import { QrCode } from '@/qr/index.js';

	let { data } = $props();

	let response = $state<any>(null);

	onMount(() => {
		pb.send('/api/compliance/send-ewc-update-start', {
			method: 'POST',
			body: {
				workflow_id: data.workflowId
			}
		})
			.then((res) => {
				response = res;
			})
			.catch((err) => {
				console.error(err);
			});
	});

	function closeConnections() {
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
	<ParamsChecker>
		{#snippet ifValid({ qr, workflowId })}
			<div>
				<h1>Wallet EWC</h1>
				<pre>{JSON.stringify(response, null, 2)}</pre>
				<QrCode src={qr} class="size-40 rounded-sm" />
			</div>
		{/snippet}
	</ParamsChecker>
</PageContent>
