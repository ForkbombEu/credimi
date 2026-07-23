<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	//

	type Props = {
		items: T[];
		maxStacked?: number;
		stackStep?: number;
		expandedStep?: number;
		getKey: (item: T) => string;
		class?: ClassValue;
		item: Snippet<[{ item: T; index: number }]>;
	};

	let {
		items,
		maxStacked = 3,
		stackStep = 4,
		expandedStep = 38,
		getKey,
		class: className = 'size-8',
		item
	}: Props = $props();

	const stacked = $derived(items.slice(0, maxStacked));
</script>

{#if stacked.length > 0}
	<div
		class={[
			'group/stack relative shrink-0 overflow-visible focus-within:z-10 hover:z-10',
			className
		]}
	>
		{#each stacked as stackItem, i (getKey(stackItem))}
			<div
				class={[
					'absolute top-0 left-0 overflow-visible focus-within:z-20 hover:z-20',
					'translate-x-(--tx-s) translate-y-(--ty-s)',
					'transition-transform duration-150 ease-out motion-reduce:transition-none',
					'group-hover/stack:translate-x-(--tx-e) group-hover/stack:translate-y-0',
					'group-focus-within/stack:translate-x-(--tx-e) group-focus-within/stack:translate-y-0'
				]}
				style:--tx-s="{-i * stackStep}px"
				style:--ty-s="{i * stackStep}px"
				style:--tx-e="{i * expandedStep}px"
				style:z-index={stacked.length - i}
			>
				{@render item({ item: stackItem, index: i })}
			</div>
		{/each}
	</div>
{/if}
