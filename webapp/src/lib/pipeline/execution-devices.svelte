<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import StatusCircle from '$lib/components/status-circle.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import type { ExecutionDevice, RunnerExecutionRef } from './live-view';

	import { runPageHref } from './execution-steps';
	import LiveViewSheet from './live-view-sheet.svelte';
	import { getExecutionDevices, type ExecutionSummary } from './workflows';

	type Props = {
		execution: ExecutionSummary;
		/** Compact layout for dense tables. */
		compact?: boolean;
		class?: string;
	};

	let { execution, compact = false, class: className = '' }: Props = $props();

	const devices = $derived(getExecutionDevices(execution));

	let sheetOpen = $state(false);
	let activeDeviceId = $state<string | undefined>(undefined);

	function openLiveView(device: ExecutionDevice) {
		activeDeviceId = device.device_id;
		sheetOpen = true;
	}

	function runnerRunHref(ref: RunnerExecutionRef): string {
		return runPageHref(ref.workflow_id, ref.run_id, ref.namespace);
	}

	const linkClass = $derived(['h-auto px-0 py-0 font-medium', compact ? 'text-xs' : 'text-sm']);
</script>

{#if devices.length === 0}
	<span class="text-muted-foreground opacity-50">—</span>
{:else}
	<ul class={['flex flex-col gap-1', className]}>
		{#each devices as device (device.device_id)}
			<li
				class={[
					'flex flex-wrap items-center gap-x-2 gap-y-0.5',
					compact ? 'text-xs' : 'text-sm'
				]}
			>
				<span class="min-w-0 truncate">{device.name}</span>
				{#if device.live_view}
					<Button
						variant="link"
						size="sm"
						class={[
							'h-auto gap-1 px-0 py-0 font-medium text-green-700',
							compact ? 'text-xs' : 'text-sm'
						]}
						onclick={() => openLiveView(device)}
					>
						<StatusCircle size={compact ? 10 : 12} />
						<span class="animate-pulse">{m.View_live()}</span>
					</Button>
				{/if}
				{#if device.preparation}
					<Button
						variant="link"
						size="sm"
						class={linkClass}
						href={runnerRunHref(device.preparation)}
					>
						{m.Device_preparation()}
					</Button>
				{/if}
				{#if device.cleanup}
					<Button
						variant="link"
						size="sm"
						class={linkClass}
						href={runnerRunHref(device.cleanup)}
					>
						{m.Device_cleanup()}
					</Button>
				{/if}
			</li>
		{/each}
	</ul>

	<LiveViewSheet
		bind:open={sheetOpen}
		workflowId={execution.execution.workflowId}
		runId={execution.execution.runId}
		title={execution.displayName}
		deviceId={activeDeviceId}
		deviceName={devices.find((d) => d.device_id === activeDeviceId)?.name}
	/>
{/if}
