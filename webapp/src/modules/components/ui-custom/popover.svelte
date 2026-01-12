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
		trigger: Snippet;
		content: Snippet;
		buttonVariants?: Parameters<typeof buttonVariants>[0];
		containerClass?: ClassValue;
		triggerClass?: ClassValue;
	};

	let {
		trigger,
		content,
		buttonVariants: buttonVariantsProps,
		containerClass,
		triggerClass
	}: Props = $props();
</script>

<Popover.Root>
	<Popover.Trigger
		class={[buttonVariants({ variant: 'outline', ...buttonVariantsProps }), triggerClass]}
	>
		{@render trigger()}
	</Popover.Trigger>
	<Popover.Content class={containerClass}>
		{@render content()}
	</Popover.Content>
</Popover.Root>
