<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { VideoIcon } from '@lucide/svelte';
	import StatusCircle from '$lib/components/status-circle.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import type { ExecutionDevice } from './live-view';

	import LiveViewSheet from './live-view-sheet.svelte';

	type Props = {
		devices: ExecutionDevice[];
		workflowId: string;
		runId: string;
		/** Compact layout for dense tables. */
		compact?: boolean;
		class?: string;
	};

	let { devices, workflowId, runId, compact = false, class: className = '' }: Props = $props();

	let sheetOpen = $state(false);
	let activeDeviceId = $state<string | undefined>(undefined);
	let activeDeviceName = $state<string | undefined>(undefined);

	function openLiveView(device: ExecutionDevice) {
		activeDeviceId = device.device_id;
		activeDeviceName = device.name;
		sheetOpen = true;
	}
</script>

{#if devices.length === 0}
	<span class="text-muted-foreground opacity-50">N/A</span>
{:else}
	<ul class={['flex flex-col gap-1', className]}>
		{#each devices as device (device.device_id)}
			<li
				class={[
					'flex flex-wrap items-center gap-x-2 gap-y-0.5',
					compact ? 'text-xs' : 'text-sm'
				]}
			>
				<span class="min-w-0 truncate font-mono">{device.name}</span>
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
						<VideoIcon class={compact ? 'size-3' : 'size-3.5'} />
						<span class="animate-pulse">{m.View_live()}</span>
					</Button>
				{/if}
			</li>
		{/each}
	</ul>

	<LiveViewSheet
		bind:open={sheetOpen}
		{workflowId}
		{runId}
		deviceId={activeDeviceId}
		deviceName={activeDeviceName}
	/>
{/if}
