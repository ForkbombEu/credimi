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
	import StackedItems from './stacked-items.svelte';

	//

	type Props = {
		items: Item[];
		class?: ClassValue;
	};

	let { items, class: className }: Props = $props();
</script>

{#if items.length === 0}
	<Na />
{:else}
	<div class={['flex items-start gap-2', className]}>
		<StackedItems {items} getKey={(item) => item.key}>
			{#snippet item({ item })}
				<EntityAvatar {item} link />
			{/snippet}
		</StackedItems>
		<List {items} layout="links-only" />
	</div>
{/if}
