<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Cog, PlayIcon } from '@lucide/svelte';
	import { Pipeline } from '$lib';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import * as ButtonGroup from '@/components/ui/button-group';
	import { m } from '@/i18n';

	import SelectModal from './device-select-modal.svelte';

	type Props = {
		pipeline: PipelinesResponse;
		onRun?: () => void;
	};

	let { pipeline, onRun }: Props = $props();

	let deviceSelectionDialogOpen = $state(false);
	let runPipelineAfterDeviceSelect = $state(false);

	const deviceType = $derived(Pipeline.Device.Binding.getType(pipeline));
	const isDeviceSpecific = $derived(deviceType === 'specific');
	const executionDevicePath = $derived(Pipeline.Device.Binding.getExecutionDevicePath(pipeline));
	const deviceRequired = $derived(Pipeline.Device.Binding.isRequired(pipeline));

	const isChecking = $derived(
		deviceRequired && !!executionDevicePath && !Pipeline.Device.Catalog.isReady()
	);

	const isDeviceOffline = $derived(
		deviceRequired &&
			Pipeline.Device.Catalog.isReady() &&
			executionDevicePath !== undefined &&
			Pipeline.Device.Catalog.findByPath(executionDevicePath)?.isOnline === false
	);

	const runDisabled = $derived(isChecking || isDeviceOffline);

	const deviceSubline = $derived.by(() => {
		const path = executionDevicePath ?? Pipeline.Device.Binding.get(pipeline.id);
		if (!path || !deviceRequired) return undefined;

		const name = path.split('/').at(-1) ?? path;

		if (isChecking) {
			return { name, status: 'checking' as const };
		}

		const offline =
			Pipeline.Device.Catalog.isReady() &&
			Pipeline.Device.Catalog.findByPath(path)?.isOnline === false;

		if (offline) {
			return { name, status: 'offline' as const };
		}

		return { name, status: undefined };
	});

	async function handleRunNow() {
		if (runDisabled) return;

		if (!deviceRequired) {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (deviceType === 'specific') {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (Pipeline.Device.Binding.get(pipeline.id)) {
			await Pipeline.run(pipeline);
			onRun?.();
			runPipelineAfterDeviceSelect = false;
			return;
		}

		runPipelineAfterDeviceSelect = true;
		deviceSelectionDialogOpen = true;
	}
</script>

{#snippet runButtonGroup()}
	<ButtonGroup.Root>
		<Button
			onclick={handleRunNow}
			disabled={runDisabled}
			class={{ 'w-[174px] justify-start': !deviceRequired }}
		>
			<PlayIcon />
			<div class="flex w-[90px] flex-col -space-y-0.5 text-left">
				<p>{m.Run_now()}</p>
				{#if deviceSubline}
					<small class="truncate text-[9px] opacity-80">
						{#if deviceSubline.status === 'checking'}
							<span class="inline-flex max-w-full items-baseline gap-0">
								<span class="shrink-0">[Checking</span>
								<span class="checking-ellipsis shrink-0" aria-hidden="true">
									<span>.</span><span>.</span><span>.</span>
								</span>
								<span class="truncate">] {deviceSubline.name}</span>
							</span>
						{:else if deviceSubline.status === 'offline'}
							[Offline] {deviceSubline.name}
						{:else}
							{deviceSubline.name}
						{/if}
					</small>
				{/if}
			</div>
		</Button>
		{#if deviceRequired}
			<IconButton
				icon={Cog}
				variant="default"
				class="rounded-none rounded-r-md border-l border-l-slate-500"
				onclick={() => (deviceSelectionDialogOpen = true)}
				disabled={isDeviceSpecific}
				tooltip={isDeviceSpecific
					? 'Device configuration is not available for pipelines with specific device steps'
					: 'Configure device'}
			/>
		{/if}
	</ButtonGroup.Root>
{/snippet}

{#if runDisabled}
	<Tooltip>
		<span class="inline-flex">
			{@render runButtonGroup()}
		</span>
		{#snippet content()}
			{#if isChecking}
				<p>{'Checking device status'}</p>
			{:else if isDeviceOffline}
				<p>{'The selected device is offline'}</p>
			{/if}
		{/snippet}
	</Tooltip>
{:else}
	{@render runButtonGroup()}
{/if}

<SelectModal
	{pipeline}
	bind:open={deviceSelectionDialogOpen}
	onSelect={() => {
		if (!runPipelineAfterDeviceSelect) return;
		void handleRunNow();
	}}
/>

<style>
	.checking-ellipsis span {
		animation: checking-dot 1.2s ease-in-out infinite;
	}

	.checking-ellipsis span:nth-child(2) {
		animation-delay: 0.2s;
	}

	.checking-ellipsis span:nth-child(3) {
		animation-delay: 0.4s;
	}

	@keyframes checking-dot {
		0%,
		20% {
			opacity: 0.2;
		}

		40%,
		100% {
			opacity: 1;
		}
	}
</style>
