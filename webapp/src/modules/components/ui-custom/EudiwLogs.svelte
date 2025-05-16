<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount } from 'svelte';
	import { Info } from 'lucide-svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { beforeNavigate } from '$app/navigation';

	let logs = $state<WorkflowLogEntry[]>([]);

	type Props = {
		workflowId: string;
		namespace: string;
	};

	const { workflowId, namespace }: Props = $props();

	onMount(() => {
		pb.realtime
			.subscribe(`${workflowId}eudiw-logs`, (data: WorkflowLogEntry[]) => {
				logs = data;
			})
			.catch((e) => {
				console.error(e);
			});

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId,
				namespace: namespace,
				signal: 'start-eudiw-check-signal'
			}
		}).catch((e) => {
			console.error(e);
		});
	});

	type WorkflowLogEntry = {
		_id: string;
		msg: string;
		src: string;
		time?: number;

		result?: 'SUCCESS' | 'ERROR' | 'FAILED' | 'WARNING' | 'INFO' | string;

		[key: string]: any;
	};

	function closeConnections() {
		pb.realtime.unsubscribe(`${workflowId}eudiw-logs`).catch((e) => {
			console.error(e);
		});

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId,
				namespace: namespace,
				signal: 'stop-eudiw-check-signal'
			}
		}).catch((e) => {
			console.error(e);
		});
		return true;
	}

	onDestroy(() => {
		closeConnections();
	});

	beforeNavigate((conf) => {
		closeConnections();
	});
</script>

<svelte:window on:beforeunload|preventDefault={closeConnections} />

<div class="py-2">
	{#if logs.length === 0}
		<Alert variant="info" icon={Info}>
			<p>Waiting for logs...</p>
		</Alert>
	{:else}
		<pre class="bg-secondary overflow-x-scroll rounded-md p-4 text-sm">
			{JSON.stringify(logs, null, 2)}
		</pre>
	{/if}
</div>
