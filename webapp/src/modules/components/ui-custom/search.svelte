<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- TODO - Convert to Svelte 5 -->
<script lang="ts" context="module">
	import type { MaybePromise } from '@/utils/types';

	export type SearchResult<T> = {
		value: T;
		label: string;
		disabled?: boolean;
		description?: string;
	};

	export type SearchFunction<T> = (text: string | undefined) => MaybePromise<SearchResult<T>[]>;
</script>

<script lang="ts" generics="T">
	import { m } from '@/i18n';
	import { createCombobox } from '@melt-ui/svelte';
	import { Check, ChevronDown, ChevronUp } from 'lucide-svelte';
	import { fly } from 'svelte/transition';
	import Spinner from './spinner.svelte';
	import Icon from './icon.svelte';
	import T from './t.svelte';
	import type { ControlAttrs } from 'formsnap';

	//

	export let searchFn: SearchFunction<T> = () => [];
	export let onSelect: (item: T) => void = () => {};

	export let label: string | undefined = undefined;
	export let placeholder = m.Search();
	export let disabled = false;

	export let controlAttrs: ControlAttrs | undefined = undefined;

	//

	const {
		elements: { menu, input, option, label: labelBuilder },
		states: { open, inputValue, touchedInput, selected },
		helpers: { isSelected }
	} = createCombobox({
		forceVisible: true
	});

	//

	let debounceTimer: ReturnType<typeof setTimeout>;

	const debounce = (callback: () => void) => {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(callback, 450);
	};

	//

	let resultsPromise: MaybePromise<SearchResult<T>[]> = searchFn(undefined);

	$: {
		if ($touchedInput) {
			debounce(() => (resultsPromise = searchFn($inputValue)));
		} else {
			resultsPromise = searchFn(undefined);
		}
	}

	//

	let searchText: string | undefined = undefined;

	$: if ($selected) {
		onSelect(($selected.value as SearchResult<T>).value);
		$selected = undefined;
		searchText = undefined;
	}
</script>

<div class="flex flex-col gap-1">
	{#if label}
		<!-- classes copied from webapp/src/modules/components/ui/label/label.svelte -->
		<!-- svelte-ignore a11y-label-has-associated-control - $label contains the 'for' attribute -->
		<label
			class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
			{...$labelBuilder}
			use:$labelBuilder.action
		>
			<span class="text-sm font-medium text-gray-900">{label}</span>
		</label>
	{/if}

	<div class="relative">
		<!-- classes copied from webapp/src/modules/components/ui/input/input.svelte -->
		<input
			{...$input}
			use:$input.action
			class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
			{placeholder}
			{disabled}
			bind:value={searchText}
			{...controlAttrs}
		/>
		<div
			class="text-secondary-foreground absolute right-3 top-1/2 z-10 -translate-y-1/2 opacity-50"
		>
			<Icon src={$open ? ChevronUp : ChevronDown} class="size-4"></Icon>
		</div>
	</div>
</div>

{#if $open}
	<ul
		class=" z-10 flex max-h-[300px] flex-col overflow-hidden rounded-md border shadow-lg"
		{...$menu}
		use:$menu.action
		transition:fly={{ duration: 150, y: -5 }}
	>
		<!-- svelte-ignore a11y-no-noninteractive-tabindex -->
		<div
			class="flex max-h-full flex-col gap-0 overflow-y-auto bg-white p-1 text-black"
			tabindex="0"
		>
			{#await resultsPromise}
				<div class="flex items-center justify-center self-stretch p-2">
					<Spinner />
				</div>
			{:then results}
				{#each results as result, index (index)}
					{@const item = result.value}
					{@const opt = $option({
						value: result,
						label: result.label,
						disabled: result.disabled
					})}

					<li
						{...opt}
						use:opt.action
						class="data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
					>
						{#if $isSelected(result)}
							<div class="check absolute left-2 top-1/2 z-10 text-gray-900">
								<Check class="size-4" />
							</div>
						{/if}
						<div>
							<slot name="item" {item}>
								<span>{result.label}</span>
								{#if result.description}
									<span class="block text-sm opacity-75">{result.description}</span>
								{/if}
							</slot>
						</div>
					</li>
				{:else}
					<li class="text-secondary-foreground/30 flex justify-center py-2">
						<T tag="small">
							{m.No_results_found()}
						</T>
					</li>
				{/each}
			{/await}
		</div>
	</ul>
{/if}

<style lang="postcss">
	.check {
		@apply absolute left-2 top-1/2 text-gray-500;
		translate: 0 calc(-50% + 1px);
	}
</style>
