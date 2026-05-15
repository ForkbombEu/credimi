<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import { getRecordByCanonifiedPath } from '$lib/canonify';
	import { getPath } from '$lib/utils';
	import { Cog, PlayIcon } from '@lucide/svelte';
	import { isError } from 'effect/Predicate';

	import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import * as ButtonGroup from '@/components/ui/button-group';
	import { m } from '@/i18n';

	import * as Runners from '../runners';
	import * as Runner from './binding';
	import SelectModal from './runner-select-modal.svelte';

	type Props = {
		pipeline: PipelinesResponse;
		onRun?: () => void;
	};

	let { pipeline, onRun }: Props = $props();

	let runnerSelectionDialogOpen = $state(false);
	let runPipelineAfterRunnerSelect = $state(false);

	const runnerType = $derived(Runner.getType(pipeline));
	const isRunnerSpecific = $derived(runnerType === 'specific');
	const executionPath = $derived(Runner.getExecutionRunnerPath(pipeline));
	const isRunnerOffline = $derived(
		executionPath !== undefined && Runners.status.isOnline(executionPath) === false
	);

	const runnerLabel = $derived.by(() => {
		const path = executionPath ?? Runner.get(pipeline.id);
		if (!path || !Runner.isRequired(pipeline)) return undefined;
		const name = path.split('/').at(-1);
		return isRunnerOffline ? `[Offline] ${name}` : name;
	});

	$effect(() => {
		const path = executionPath;
		if (!path || Runners.status.isOnline(path) !== undefined) return;

		const fromStore = Runners.store.read().find((r) => getPath(r) === path);
		if (fromStore) {
			Runners.status.probe([fromStore], { reason: 'visible' });
			return;
		}

		let cancelled = false;
		void getRecordByCanonifiedPath<MobileRunnersResponse>(path)
			.then((res) => {
				if (cancelled || isError(res)) return;
				Runners.status.probe([res], { reason: 'visible' });
			})
			.catch(console.error);

		return () => {
			cancelled = true;
		};
	});

	async function handleRunNow() {
		if (isRunnerOffline) return;

		if (!Runner.isRequired(pipeline)) {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (runnerType === 'specific') {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (Runner.get(pipeline.id)) {
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
			disabled={isRunnerOffline}
			class={{ 'w-[174px] justify-start': !Runner.isRequired(pipeline) }}
		>
			<PlayIcon />
			<div class="flex w-[90px] flex-col -space-y-0.5 text-left">
				<p>{m.Run_now()}</p>
				{#if runnerLabel}
					<small class="truncate text-[9px] opacity-80">{runnerLabel}</small>
				{/if}
			</div>
		</Button>
		{#if Runner.isRequired(pipeline)}
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

{#if isRunnerOffline}
	<Tooltip>
		<span class="inline-flex">
			{@render runButtonGroup()}
		</span>
		{#snippet content()}
			<p>{m.Runner_offline_run_disabled()}</p>
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
