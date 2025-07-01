<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount, onDestroy, type ComponentProps } from 'svelte';
	import { beforeNavigate } from '$app/navigation';
	import {
		createWorkflowLogHandlers,
		LogStatus,
		type WorkflowLog,
		type WorkflowLogsProps
	} from './workflow-logs.js';
	import { Info } from 'lucide-svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Badge } from '@/components/ui/badge/index.js';
	import * as Accordion from '@/components/ui/accordion/index.js';
	import { m } from '@/i18n/index.js';
	import { nanoid } from 'nanoid';

	const props: WorkflowLogsProps & { class?: string; uiSize?: 'sm' | 'md' } = $props();

	let logs: WorkflowLog[] = $state([]);

	const { startLogs, stopLogs } = createWorkflowLogHandlers({
		...props,
		onUpdate: (data) => {
			logs = data.reverse();
		}
	});

	onMount(startLogs);
	onDestroy(stopLogs);
	beforeNavigate(() => {
		stopLogs();
	});

	//

	type BadgeVariant = ComponentProps<typeof Badge>['variant'];

	function statusToVariant(status: LogStatus): BadgeVariant {
		switch (status) {
			case LogStatus.SUCCESS:
				return 'default';
			case LogStatus.ERROR:
				return 'destructive';
			case LogStatus.FAILED:
				return 'destructive';
			case LogStatus.FAILURE:
				return 'destructive';
			default:
				return 'outline';
		}
	}
</script>

<svelte:window on:beforeunload|preventDefault={stopLogs} />

{#if logs.length === 0}
	<Alert variant="info" icon={Info}>
		<p>{m.Waiting_for_logs()}</p>
	</Alert>
{:else}
	<div class={['max-h-[700px] space-y-1 overflow-y-auto', props.class]}>
		{#each logs as log}
			{@const logId = nanoid(4)}
			{@const status = log.status ?? LogStatus.INFO}
			<Accordion.Root type="multiple" class="bg-muted space-y-1 rounded-md px-2">
				<Accordion.Item value={logId} class="border-none">
					<Accordion.Trigger
						class="flex items-center justify-between gap-2 hover:no-underline"
					>
						<div class="flex grow items-center gap-2">
							<Badge
								class="w-20 text-center capitalize"
								variant={statusToVariant(status)}
							>
								{status}
							</Badge>

							{#if log.message}
								<p
									class={[
										'text-left',
										{
											'text-sm': props.uiSize === 'sm',
											'text-md': props.uiSize === 'md'
										}
									]}
								>
									{log.message}
								</p>
							{:else}
								<p class="text-muted-foreground">{m.Open_for_details()}</p>
							{/if}
						</div>

						{#if log.time}
							<p
								class="text-muted-foreground shrink-0 text-nowrap text-right text-xs"
							>
								{new Date(log.time).toLocaleString()}
							</p>
						{/if}
					</Accordion.Trigger>

					<Accordion.Content>
						<pre
							class="bg-secondary overflow-x-scroll rounded-md p-2 text-xs">{JSON.stringify(
								log.rawLog,
								null,
								2
							)}</pre>
					</Accordion.Content>
				</Accordion.Item>
			</Accordion.Root>
		{/each}
	</div>
{/if}
