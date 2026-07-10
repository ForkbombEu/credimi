<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { EmptyState, WithEmptyState } from '$lib/pipeline-form/steps/_partials/index.js';
	import { fly } from 'svelte/transition';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { RunnerSelectPresentation } from './runner-select-catalog.svelte.js';
	import type { RunnerRecord } from './types';

	import RunnerSelectListItem from './runner-select-list-item.svelte';

	type Props = {
		presentation: RunnerSelectPresentation;
		foundRunners: RunnerRecord[];
		catalogLoading: boolean;
		onSelect?: (runner: RunnerRecord) => void;
		selectedRunner?: string;
		scrollable?: boolean;
		prepend?: Snippet;
		emptyContainerClass?: ClassValue;
		listContainerClass?: ClassValue;
	};

	let {
		presentation,
		foundRunners,
		catalogLoading,
		onSelect,
		selectedRunner,
		scrollable = false,
		prepend,
		emptyContainerClass = 'p-0!',
		listContainerClass
	}: Props = $props();
</script>

{#if catalogLoading}
	<EmptyState containerClass={emptyContainerClass}>
		<Spinner size={16} />
		<T>{m.Loading()}</T>
	</EmptyState>
{:else}
	<div class="space-y-1" transition:fly>
		{@render prepend?.()}

		{#if scrollable}
			<WithEmptyState
				items={foundRunners}
				emptyText={m.No_runners_found()}
				containerClass={listContainerClass}
			>
				{#snippet item({ item })}
					<RunnerSelectListItem {item} {presentation} {selectedRunner} {onSelect} />
				{/snippet}
			</WithEmptyState>
		{:else}
			<div class={['space-y-2', listContainerClass]}>
				{#each foundRunners as item (item.path)}
					<RunnerSelectListItem {item} {presentation} {selectedRunner} {onSelect} />
				{:else}
					<EmptyState text={m.No_runners_found()} containerClass={emptyContainerClass} />
				{/each}
			</div>
		{/if}
	</div>
{/if}
