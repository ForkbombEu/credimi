<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type ItemCardSnippets = {
		left?: () => ReturnType<Snippet>;
		children?: Snippet<[{ Title: typeof Title; Description: typeof Description }]>;
		right?: () => ReturnType<Snippet>;
	};
</script>

<script lang="ts">
	import Title from './itemCardTitle.svelte';
	import Description from './itemCardDescription.svelte';
	import type { Snippet } from 'svelte';
	import type { BitsDivAttributes } from 'bits-ui';

	type Props = Omit<BitsDivAttributes, 'children'> & ItemCardSnippets;

	let { class: className = '', left, children, right, ...rest }: Props = $props();
</script>

<div class="flex items-center gap-4 rounded-lg border p-4 {className}" {...rest}>
	{#if left}
		<div class="shrink-0">
			{@render left?.()}
		</div>
	{/if}

	<div class="w-0 grow overflow-hidden">
		{@render children?.({ Title, Description })}
	</div>

	{#if right}
		<div class="shrink-0">
			{@render right?.()}
		</div>
	{/if}
</div>
