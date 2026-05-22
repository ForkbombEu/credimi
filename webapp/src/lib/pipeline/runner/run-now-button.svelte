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

	import SelectModal from './runner-select-modal.svelte';

	type Props = {
		pipeline: PipelinesResponse;
		onRun?: () => void;
	};

	let { pipeline, onRun }: Props = $props();

	let runnerSelectionDialogOpen = $state(false);
	let runPipelineAfterRunnerSelect = $state(false);

	const runnerType = $derived(Pipeline.Runner.Binding.getType(pipeline));
	const isRunnerSpecific = $derived(runnerType === 'specific');
	const executionPath = $derived(Pipeline.Runner.Binding.getExecutionRunnerPath(pipeline));
	const runnerRequired = $derived(Pipeline.Runner.Binding.isRequired(pipeline));

	const isChecking = $derived(
		runnerRequired && !!executionPath && !Pipeline.Runner.Catalog.isReady()
	);

	const isRunnerOffline = $derived(
		runnerRequired &&
			Pipeline.Runner.Catalog.isReady() &&
			executionPath !== undefined &&
			Pipeline.Runner.Catalog.findByPath(executionPath)?.isOnline === false
	);

	const runDisabled = $derived(isChecking || isRunnerOffline);

	const runnerSubline = $derived.by(() => {
		const path = executionPath ?? Pipeline.Runner.Binding.get(pipeline.id);
		if (!path || !runnerRequired) return undefined;

		const name = path.split('/').at(-1) ?? path;

		if (isChecking) {
			return { name, status: 'checking' as const };
		}

		const offline =
			Pipeline.Runner.Catalog.isReady() &&
			Pipeline.Runner.Catalog.findByPath(path)?.isOnline === false;

		if (offline) {
			return { name, status: 'offline' as const };
		}

		return { name, status: undefined };
	});

	async function handleRunNow() {
		if (runDisabled) return;

		if (!runnerRequired) {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (runnerType === 'specific') {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (Pipeline.Runner.Binding.get(pipeline.id)) {
			await Pipeline.run(pipeline);
			onRun?.();
			runPipelineAfterRunnerSelect = false;
			return;
		}

		runPipelineAfterRunnerSelect = true;
		runnerSelectionDialogOpen = true;
	}
</script>

{#snippet runButtonGroup()}
	<ButtonGroup.Root>
		<Button
			onclick={handleRunNow}
			disabled={runDisabled}
			class={{ 'w-[174px] justify-start': !runnerRequired }}
		>
			<PlayIcon />
			<div class="flex w-[90px] flex-col -space-y-0.5 text-left">
				<p>{m.Run_now()}</p>
				{#if runnerSubline}
					<small class="truncate text-[9px] opacity-80">
						{#if runnerSubline.status === 'checking'}
							<span class="inline-flex max-w-full items-baseline gap-0">
								<span class="shrink-0">[Checking</span>
								<span class="checking-ellipsis shrink-0" aria-hidden="true">
									<span>.</span><span>.</span><span>.</span>
								</span>
								<span class="truncate">] {runnerSubline.name}</span>
							</span>
						{:else if runnerSubline.status === 'offline'}
							[Offline] {runnerSubline.name}
						{:else}
							{runnerSubline.name}
						{/if}
					</small>
				{/if}
			</div>
		</Button>
		{#if runnerRequired}
			<IconButton
				icon={Cog}
				variant="default"
				class="rounded-none rounded-r-md border-l border-l-slate-500"
				onclick={() => (runnerSelectionDialogOpen = true)}
				disabled={isRunnerSpecific}
				tooltip={isRunnerSpecific
					? m.Runner_configuration_not_available()
					: m.Configure_runner()}
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
				<p>{m.Runner_status_checking()}</p>
			{:else if isRunnerOffline}
				<p>{m.Runner_offline_run_disabled()}</p>
			{/if}
		{/snippet}
	</Tooltip>
{:else}
	{@render runButtonGroup()}
{/if}

<SelectModal
	{pipeline}
	bind:open={runnerSelectionDialogOpen}
	onSelect={() => {
		if (!runPipelineAfterRunnerSelect) return;
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
