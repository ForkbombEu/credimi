<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { m } from '@/i18n';

	import LiveViewFrame from './live-view-frame.svelte';

	type Props = {
		open?: boolean;
		workflowId: string;
		runId: string;
		title: string;
		deviceId: string | undefined;
		deviceName?: string;
	};

	let {
		open = $bindable(false),
		workflowId,
		runId,
		title,
		deviceId,
		deviceName
	}: Props = $props();
</script>

<Sheet
	bind:open
	hideTrigger
	title={m.Live_view()}
	class="!w-[min(100vw,48rem)]"
	contentClass="flex min-h-0 flex-1 flex-col gap-4"
>
	{#snippet content()}
		{#if open && deviceId && workflowId && runId}
			<div class="space-y-3 text-sm">
				<div>
					<p class="font-semibold">{title}</p>
					{#if deviceName}
						<p>{m.Device()}: {deviceName}</p>
					{/if}
				</div>
			</div>

			<LiveViewFrame
				{workflowId}
				{runId}
				{deviceId}
				{deviceName}
				class="h-[70dvh] min-h-96"
			/>

			<dl class="space-y-3 text-sm text-muted-foreground">
				<div>
					<dt>{m.Workflow_ID()}</dt>
					<dd class="truncate font-mono text-xs">{workflowId}</dd>
				</div>
				<div>
					<dt>Run ID</dt>
					<dd class="truncate font-mono text-xs">{runId}</dd>
				</div>
			</dl>
		{/if}
	{/snippet}
</Sheet>
