<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Record } from '$lib/pipeline/runner';

	import { browser } from '$app/environment';
	import { Pipeline } from '$lib';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import Dialog from '@/components/ui-custom/dialog.svelte';
	import { m } from '@/i18n';

	import RunnerSelectInput from './runner-select-input.svelte';

	//

	type Props = {
		open?: boolean;
		pipeline: PipelinesResponse;
		title?: string;
		description?: string;
		onSelect?: (runner: Record) => void;
	};

	let {
		open = $bindable(false),
		pipeline,
		title = m.Select_runner(),
		description = m.Select_a_runner_to_execute_the_pipeline(),
		onSelect
	}: Props = $props();

	//

	function handleSelect(runner: Record) {
		Pipeline.Runner.Binding.set(pipeline, runner);
		open = false;
		onSelect?.(runner);
	}

	//

	let currentRunnerPath = $derived.by(() => {
		if (!browser) return undefined;
		return Pipeline.Runner.Binding.get(pipeline.id);
	});

	// let currentRunner = $derived.by(() => {
	// 	if (!currentRunnerPath) return undefined;
	// 	Pipeline.Runner.Catalog.read();
	// 	return Pipeline.Runner.Catalog.findByPath(currentRunnerPath);
	// });

	$effect(() => {
		if (!open) return;
		void Pipeline.Runner.Catalog.refresh();
	});
</script>

<Dialog bind:open {title} {description} hideTrigger>
	{#snippet content()}
		<!-- {#if currentRunner}
			<Alert variant="info" class="bg-blue-50">
				<T>
					<span>{m.Current_runner()}:</span>
					<span class="font-semibold">{currentRunner.name} </span>
				</T>
			</Alert>
		{/if} -->
		<RunnerSelectInput
			presentation="run"
			onSelect={handleSelect}
			selectedRunner={currentRunnerPath}
		/>
	{/snippet}
</Dialog>
