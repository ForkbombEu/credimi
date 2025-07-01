<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount, type Snippet } from 'svelte';
	import Container from './container.svelte';

	type Props = {
		workflowId: string;
		runId: string;
		namespace: string;
		children?: Snippet;
	};

	let { workflowId, runId, namespace, children }: Props = $props();

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

<Container>
	{#snippet left()}
		{@render children?.()}
	{/snippet}
</Container>
