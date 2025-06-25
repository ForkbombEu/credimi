<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import {
		createWorkflowLogHandlers,
		type WorkflowLogEntry,
		type WorkflowLogsProps
	} from './workflow-logs.js';
	import { Info } from 'lucide-svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Badge } from '@/components/ui/badge/index.js';
	import * as Accordion from '@/components/ui/accordion/index.js';
	import { m } from '@/i18n/index.js';

	//

	const props: WorkflowLogsProps = $props();

	let logs: WorkflowLogEntry[] = $state([]);

	const { startLogs, stopLogs } = $derived(
		createWorkflowLogHandlers({
			...props,
			onUpdate: (data: WorkflowLogEntry[]) => {
				logs = data.reverse();
			}
		})
	);

	$effect(() => {
		startLogs();
		return () => stopLogs();
	});

	beforeNavigate(() => {
		stopLogs();
	});
</script>

<svelte:window on:beforeunload|preventDefault={stopLogs} />

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
