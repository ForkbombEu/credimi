<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, onMount } from 'svelte';
	import { Info } from 'lucide-svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Badge } from '../ui/badge/index.js';
	import * as Accordion from '../ui/accordion/index.js';

	let logs = $state<WorkflowLogEntry[]>([]);

	type Props = {
		workflowId: string;
		namespace: string;
	};

	const { workflowId, namespace }: Props = $props();

	onMount(() => {
		pb.realtime
			.subscribe(`${workflowId}openidnet-logs`, (data: WorkflowLogEntry[]) => {
				console.log(data);
				logs = data;
			})
			.catch((e) => {
				console.error(e);
			});

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId + '-log',
				namespace: namespace,
				signal: 'start-openidnet-check-log-update'
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
		pb.realtime.unsubscribe(`${workflowId}openidnet-logs`).catch((e) => {
			console.error(e);
		});

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId + '-log',
				namespace: namespace,
				signal: 'stop-openidnet-check-log-update'
			}
		}).catch((e) => {
			console.error(e);
		});
	}

	onDestroy(() => {
		closeConnections();
	});
</script>

<svelte:window on:beforeunload={closeConnections} />

<div class="flex flex-col gap-2 py-2">
	{#if logs.length === 0}
		<Alert variant="info" icon={Info}>
			<p>Waiting for logs...</p>
		</Alert>
	{:else}
		{#each logs as log}
			<Accordion.Root type="multiple" class="bg-muted rounded-md px-2">
				<Accordion.Item value={log._id} class="border-none">
					<Accordion.Trigger
						class="flex flex-row items-center justify-start gap-2 hover:no-underline"
					>
						{#if log.result}
							<Badge
								variant={log.result === 'SUCCESS'
									? 'default'
									: log.result === 'ERROR'
										? 'destructive'
										: 'outline'}
							>
								{log.result}
							</Badge>
						{/if}
						<span>{log.msg}</span>
						{#if log.time}
							<p class="text-muted-foreground text-xs">
								{new Date(log.time).toLocaleString()}
							</p>
						{/if}
					</Accordion.Trigger>
					<Accordion.Content>
						<pre
							class="bg-secondary overflow-x-scroll rounded-md p-2 text-xs">{JSON.stringify(
								log,
								null,
								2
							)}</pre>
					</Accordion.Content>
				</Accordion.Item>
			</Accordion.Root>
		{/each}
	{/if}
</div>
