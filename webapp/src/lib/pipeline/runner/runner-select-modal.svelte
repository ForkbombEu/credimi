<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { MobileRunnerListItem } from '../runners/utils';

	import * as Runners from '../runners';
	import * as Runner from './binding';
	import RunnerSelectInput from './runner-select-input.svelte';

	//

	type Props = {
		open?: boolean;
		pipeline: PipelinesResponse;
		title?: string;
		description?: string;
		onSelect?: (runner: MobileRunnerListItem) => void;
	};

	let {
		open = $bindable(false),
		pipeline,
		title = m.Select_runner(),
		description = m.Select_a_runner_to_execute_the_pipeline(),
		onSelect
	}: Props = $props();

	//

	function handleSelect(runner: MobileRunnerListItem) {
		Runner.set(pipeline, runner);
		currentRunnerPath = runner.runner_id;
		currentRunner = runner;
		open = false;
		onSelect?.(runner);
	}

	//

	let currentRunnerPath = $derived.by(() => {
		if (!browser) return undefined;
		return Runner.get(pipeline.id);
	});

	let currentRunner = $state<MobileRunnerListItem>();

	$effect(() => {
		if (!currentRunnerPath) return;
		currentRunner = Runners.store
			.read()
			.find((runner) => runner.runner_id === currentRunnerPath);
	});

	$effect(() => {
		if (!open) return;
		Runners.status.probe(Runners.store.read(), { reason: 'modal' });
	});
</script>

<Dialog bind:open {title} {description} hideTrigger>
	{#snippet content()}
		{#if currentRunner}
			<Alert variant="info" class="bg-blue-50">
				<T>
					<span>{m.Current_runner()}:</span>
					<span class="font-semibold">{currentRunner.name} </span>
				</T>
			</Alert>
		{/if}
		<RunnerSelectInput onSelect={handleSelect} />
	{/snippet}
</Dialog>
