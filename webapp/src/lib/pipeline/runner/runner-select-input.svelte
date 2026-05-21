<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
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
	};

	let {
		presentation = 'picker',
		onSelect,
		selectedRunner,
		name,
		required = false
	}: Props = $props();

	const runnerCatalog = bindRunnerCatalogSearch();
</script>

{#if runnerCatalog.catalogLoading}
	<RunnerSelectList
		{presentation}
		foundRunners={runnerCatalog.foundRunners}
		catalogLoading={runnerCatalog.catalogLoading}
		{onSelect}
		{selectedRunner}
	/>
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

		<RunnerSelectList
			{presentation}
			foundRunners={runnerCatalog.foundRunners}
			catalogLoading={runnerCatalog.catalogLoading}
			{onSelect}
			{selectedRunner}
		/>
	</div>
{/if}
