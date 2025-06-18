<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beforeNavigate } from '$app/navigation';
	import { createWorkflowLogHandlers, type WorkflowLogEntry } from './logic.js';
	import { Info } from 'lucide-svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Badge } from '@/components/ui/badge/index.js';
	import * as Accordion from '@/components/ui/accordion/index.js';
	import { m } from '@/i18n/index.js';

	type Props = {
		workflowId: string;
		namespace: string;
		subscriptionSuffix: string;
		startSignal: string;
		stopSignal: string;
		workflowSignalSuffix?: string;
	};
	const {
		workflowId,
		namespace,
		subscriptionSuffix,
		startSignal,
		stopSignal,
		workflowSignalSuffix
	}: Props = $props();

	let logs: WorkflowLogEntry[] = $state([]);

	const { onMount: mountLogs, onDestroy: destroyLogs } = createWorkflowLogHandlers({
		workflowId,
		namespace,
		subscriptionSuffix,
		workflowSignalSuffix,
		startSignal,
		stopSignal,
		onUpdate: (data: WorkflowLogEntry[]) => {
			logs = data;
		}
	});

	onMount(mountLogs);
	onDestroy(destroyLogs);
	beforeNavigate(() => {
		destroyLogs();
	});
</script>

<svelte:window on:beforeunload|preventDefault={destroyLogs} />

<div class="py-2">
	{#if logs.length === 0}
		<Alert variant="info" icon={Info}>
			<p>{m.Waiting_for_logs()}</p>
		</Alert>
	{:else}
		{#each logs as log}
			<Accordion.Root type="multiple" class="bg-muted rounded-md px-2">
				<Accordion.Item value={log._id} class="border-none">
					<Accordion.Trigger class="flex items-center gap-2 hover:no-underline">
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
						<pre class="bg-secondary overflow-x-scroll rounded-md p-2 text-xs">
{JSON.stringify(log, null, 2)}
            </pre>
					</Accordion.Content>
				</Accordion.Item>
			</Accordion.Root>
		{/each}
	{/if}
</div>
