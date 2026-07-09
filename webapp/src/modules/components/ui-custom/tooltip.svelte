<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import * as Tooltip from '@/components/ui/tooltip/index.js';

	type Props = {
		children?: Snippet;
		child?: Snippet<[{ props: Record<string, unknown> }]>;
		content?: Snippet;
		delayDuration?: number;
		disabled?: boolean;
		contentClass?: string;
	};

	let {
		children,
		child,
		content,
		delayDuration = 500,
		disabled = false,
		contentClass
	}: Props = $props();
</script>

{#if disabled}
	{@render children?.()}
	{@render child?.({ props: {} })}
{:else}
	<Tooltip.Provider {delayDuration}>
		<Tooltip.Root>
			{#if child}
				<Tooltip.Trigger {child} />
			{:else if children}
				<Tooltip.Trigger>{@render children()}</Tooltip.Trigger>
			{/if}

			<Tooltip.Content class={contentClass}>
				{@render content?.()}
			</Tooltip.Content>
		</Tooltip.Root>
	</Tooltip.Provider>
{/if}
