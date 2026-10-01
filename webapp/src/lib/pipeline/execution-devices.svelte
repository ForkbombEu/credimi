<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import StatusCircle from '$lib/components/status-circle.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import type { ExecutionDevice } from './live-view';

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
			</li>
		{/each}
	</ul>

	<LiveViewSheet bind:open={sheetOpen} {execution} deviceId={activeDeviceId} />
{/if}
