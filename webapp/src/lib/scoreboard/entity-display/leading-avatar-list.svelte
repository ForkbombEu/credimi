<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import type { Item } from './types';

	import EntityAvatar from './avatar.svelte';
	import List from './list.svelte';
	import Na from './na.svelte';

	//

	const MAX_STACKED = 3;
	/** Pixels left/down between piled avatars. */
	const STACK_STEP = 4;
	/** Pixels between avatars when expanded (size-8 + gap). */
	const EXPANDED_STEP = 38;

	type Props = {
		items: Item[];
		class?: ClassValue;
	};

	let { items, class: className }: Props = $props();

	const stacked = $derived(items.slice(0, MAX_STACKED));
</script>

{#if stacked.length === 0}
	<Na />
{:else}
	<div class={['flex items-start gap-2', className]}>
		<div
			class="group/stack relative size-8 shrink-0 overflow-visible focus-within:z-10 hover:z-10"
		>
			{#each stacked as item, i (item.key)}
				<div
					class={[
						'absolute top-0 left-0 overflow-visible focus-within:z-20 hover:z-20',
						'translate-x-(--tx-s) translate-y-(--ty-s)',
						'transition-transform duration-150 ease-out motion-reduce:transition-none',
						'group-hover/stack:translate-x-(--tx-e) group-hover/stack:translate-y-0',
						'group-focus-within/stack:translate-x-(--tx-e) group-focus-within/stack:translate-y-0'
					]}
					style:--tx-s="{-i * STACK_STEP}px"
					style:--ty-s="{i * STACK_STEP}px"
					style:--tx-e="{i * EXPANDED_STEP}px"
					style:z-index={stacked.length - i}
				>
					<EntityAvatar {item} link />
				</div>
			{/each}
		</div>
		<List {items} layout="links-only" />
	</div>
{/if}
