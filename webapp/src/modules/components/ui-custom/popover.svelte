<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { buttonVariants } from '@/components/ui/button';
	import * as Popover from '@/components/ui/popover';

	//

	type Props = {
		triggerContent?: Snippet;
		trigger?: Snippet<[{ props: Record<string, unknown> }]>;
		content: Snippet;
		buttonVariants?: Parameters<typeof buttonVariants>[0];
		containerClass?: ClassValue;
		triggerClass?: ClassValue;
	};

	let {
		triggerContent,
		trigger,
		content,
		buttonVariants: buttonVariantsProps,
		containerClass,
		triggerClass
	}: Props = $props();

	//

	const classes: ClassValue = $derived([
		buttonVariants({ variant: 'outline', ...buttonVariantsProps }),
		triggerClass
	]);
</script>

<Popover.Root>
	{#if trigger}
		<Popover.Trigger class={classes}>
			{#snippet child({ props })}
				{@render trigger({ props })}
			{/snippet}
		</Popover.Trigger>
	{:else}
		<Popover.Trigger class={classes}>
			{@render triggerContent?.()}
		</Popover.Trigger>
	{/if}

	<Popover.Content class={containerClass}>
		{@render content()}
	</Popover.Content>
</Popover.Root>
