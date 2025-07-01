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
			logs = data;
			const container = document.getElementById(containerId);
			if (!container) return;
			if (accordionValue?.length !== 0) return;
			container.scrollTo({
				top: container.scrollHeight,
				behavior: 'smooth'
			});
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

	//

	let accordionValue = $state<string>();
	const containerId = 'container' + nanoid(4);
</script>

<svelte:window on:beforeunload|preventDefault={stopLogs} />

{#if logs.length === 0}
	<Alert variant="info" icon={Info}>
		<p>{m.Waiting_for_logs()}</p>
	</Alert>
{:else}
	<div id={containerId} class={['max-h-[700px] space-y-1 overflow-y-auto', props.class]}>
		<Accordion.Root
			bind:value={accordionValue}
			type="single"
			class="flex w-full flex-col gap-1"
		>
			{#each logs as log}
				{@const status = log.status ?? LogStatus.INFO}
				<Accordion.Item class="bg-background rounded-md border-none px-2">
					<Accordion.Trigger
						class="flex items-center justify-between gap-2 hover:no-underline"
					>
						<Badge
							class="w-20 text-center capitalize"
							variant={statusToVariant(status)}
						>
							{status}
						</Badge>

						<div class="flex w-0 grow items-center gap-2">
							{#if log.message}
								<p
									class={[
										'overflow-hidden text-left',
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
						<div
							class="bg-secondary flex w-full gap-2 overflow-x-scroll rounded-md p-2"
						>
							<div class="w-0 grow">
								<pre class="text-xs">{JSON.stringify(log.rawLog, null, 2)}</pre>
							</div>
						</div>
					</Accordion.Content>
				</Accordion.Item>
			{/each}
		</Accordion.Root>
	</div>
{/if}
