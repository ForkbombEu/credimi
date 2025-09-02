<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';

	import { Array as A } from 'effect';

	interface Props {
		data?: T | T[] | undefined;
		children?: Snippet<[{ item: T; removeItem: () => void }]>;
	}

	let { data = $bindable(undefined), children }: Props = $props();

	function removeItemFromArray(item: T) {
		if (!Array.isArray(data)) return;
		data = A.remove(data, data.indexOf(item));
	}

	function removeItem() {
		if (Array.isArray(data)) return;
		data = undefined;
	}

	function typeCast(data: unknown): T {
		return data as T;
	}
</script>

{#if Array.isArray(data)}
	{#each data as item (item)}
		{@render children?.({ item, removeItem: () => removeItemFromArray(item) })}
	{/each}
{:else if data}
	{@render children?.({ item: typeCast(data), removeItem })}
{/if}
