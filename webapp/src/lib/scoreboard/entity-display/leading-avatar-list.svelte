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

	type Props = {
		items: Item[];
		class?: ClassValue;
	};

	let { items, class: className }: Props = $props();

	const first = $derived(items[0]);
</script>

{#if !first}
	<Na />
{:else}
	<div class={['flex items-start gap-2', className]}>
		<EntityAvatar item={first} link />
		<List {items} layout="links-only" />
	</div>
{/if}
