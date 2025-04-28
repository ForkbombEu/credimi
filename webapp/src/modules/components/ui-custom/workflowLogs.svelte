<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { onDestroy, untrack } from 'svelte';
	import { Info } from 'lucide-svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Badge } from '../ui/badge/index.js';
	import * as Accordion from '../ui/accordion/index.js';

	let logs = $state<WorkflowLogEntry[]>([]);

	type Props = {
		workflowId: string;
	};

	const { workflowId }: Props = $props();

	$effect(() => {
		untrack(async () => {
			try {
				const result = await pb.send('/api/compliance/check/send-log-update-start', {
					method: 'POST',
					body: {
						workflow_id: workflowId
					}
				});
				if (result) {
					pb.realtime.subscribe(
						`${workflowId}openid4vp-wallet-logs`,
						(data: WorkflowLogEntry[]) => {
							console.log(data);
							logs = data;
						}
					);
				}
			} catch (error) {
				console.error(error);
			}
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

	onDestroy(() => {
		pb.realtime.unsubscribe(`${workflowId}openid4vp-wallet-logs`);
	});
</script>

<div class="flex flex-col gap-2 py-2">
	{#if logs.length === 0}
		<Alert variant="info" icon={Info}>
			<p>Waiting for logs...</p>
		</Alert>
	{:else}
		{#each logs as log}
			<Accordion.Root type="multiple" class="rounded-md bg-muted px-2">
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
							<p class="text-xs text-muted-foreground">
								{new Date(log.time).toLocaleString()}
							</p>
						{/if}
					</Accordion.Trigger>
					<Accordion.Content>
						<pre
							class="overflow-x-scroll rounded-md bg-secondary p-2 text-xs">{JSON.stringify(
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
