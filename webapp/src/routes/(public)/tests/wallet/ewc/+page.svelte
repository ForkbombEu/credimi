<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount } from 'svelte';

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

<div>
	<h1>Wallet EWC</h1>
	<pre>{JSON.stringify(response, null, 2)}</pre>
</div>
