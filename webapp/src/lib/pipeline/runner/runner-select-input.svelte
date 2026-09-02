<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SearchInput } from '$lib/pipeline-form/steps/_partials/index.js';
	import { fly } from 'svelte/transition';

	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n';

	import type { RunnerRecord } from './types';

	import { bindRunnerCatalogSearch } from './runner-select-catalog.svelte.js';
	import RunnerSelectList from './runner-select-list.svelte';

	//

	type Presentation = 'picker' | 'run';

	type Props = {
		presentation?: Presentation;
		onSelect?: (runner: RunnerRecord) => void;
		selectedRunner?: string;
		name?: string;
		required?: boolean;
		constrainResults?: boolean;
	};

	let {
		presentation = 'picker',
		onSelect,
		selectedRunner,
		name,
		required = false,
		constrainResults = false
	}: Props = $props();

	const runnerCatalog = bindRunnerCatalogSearch();
</script>

{#snippet runnerResults()}
	<RunnerSelectList
		{presentation}
		foundRunners={runnerCatalog.foundRunners}
		catalogLoading={runnerCatalog.catalogLoading}
		{onSelect}
		{selectedRunner}
	/>
{/snippet}

{#if runnerCatalog.catalogLoading}
	{@render runnerResults()}
{:else}
	<div class="space-y-3" transition:fly>
		<div class="space-y-2">
			<Label for={name}>
				{m.Runner()}
				{#if required}
					<span class="font-bold text-destructive">*</span>
				{/if}
			</Label>
			<SearchInput search={runnerCatalog.runnerSearch} {name} />
		</div>

		{#if constrainResults}
			<div class="max-h-64 overflow-y-auto pr-2" role="region" aria-label={m.Runners()}>
				{@render runnerResults()}
			</div>
			{#if runnerCatalog.foundRunners.length > 5}
				<p class="text-xs text-muted-foreground" aria-live="polite">
					{m.runner_picker_scroll_hint({ count: runnerCatalog.foundRunners.length })}
				</p>
			{/if}
		{:else}
			{@render runnerResults()}
		{/if}
	</div>
{/if}
