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
		pb.send('/api/...', {
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

	onDestroy(() => {
		pb.send('/api/...', {
			method: 'POST',
			body: {
				workflow_id: data.workflowId
			}
		});
	});
</script>

<div>
	<h1>Wallet EWC</h1>
</div>
