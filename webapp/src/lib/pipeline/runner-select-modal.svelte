<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';
	import { getRecordByCanonifiedPath } from '$lib/canonify';
	import { getPath } from '$lib/utils';
	import { isError } from 'effect/Predicate';

	import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import * as PipelineRunner from './runner';
	import RunnerSelectInput from './runner-select-input.svelte';

	//

	type Props = {
		open?: boolean;
		pipeline: PipelinesResponse;
		title?: string;
		description?: string;
		onSelect?: (runner: MobileRunnersResponse) => void;
	};

	let {
		open = $bindable(false),
		pipeline,
		title = m.Select_runner(),
		description = m.Select_a_runner_to_execute_the_pipeline(),
		onSelect
	}: Props = $props();

	//

	function handleSelect(runner: MobileRunnersResponse) {
		PipelineRunner.set(pipeline, runner);
		currentRunnerPath = getPath(runner);
		currentRunner = runner;
		open = false;
		onSelect?.(runner);
	}

	//

	let currentRunnerPath = $derived.by(() => {
		if (!browser) return undefined;
		return PipelineRunner.get(pipeline.id);
	});

	let currentRunner = $state<MobileRunnersResponse>();

	$effect(() => {
		if (!currentRunnerPath) return;
		getRecordByCanonifiedPath<MobileRunnersResponse>(currentRunnerPath)
			.then((res) => {
				if (isError(res)) throw res;
				else currentRunner = res;
			})
			.catch((e) => {
				console.error(e);
			});
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
