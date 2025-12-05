<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type DropdownMenuItem = {
		label: string;
		icon?: IconComponent;
		onclick?: () => void;
		disabled?: boolean;
	};
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import * as DropdownMenu from '@/components/ui/dropdown-menu';

	import type { IconComponent } from '../types';

	import { buttonVariants } from '../ui/button';
	import Icon from './icon.svelte';

	//

	type Props = {
		title?: string;
		trigger?: Snippet;
		buttonVariants?: Parameters<typeof buttonVariants>[0];
		containerClass?: ClassValue;
		items: DropdownMenuItem[];
	};

	let {
		title,
		trigger,
		buttonVariants: buttonVariantsProps,
		containerClass,
		items
	}: Props = $props();
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger class={buttonVariants({ variant: 'outline', ...buttonVariantsProps })}>
		{@render trigger?.()}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content class={containerClass}>
		<DropdownMenu.Group>
			{#if title}
				<DropdownMenu.Label>{title}</DropdownMenu.Label>
				<DropdownMenu.Separator />
			{/if}

			{#each items as item (item.label)}
				<DropdownMenu.Item
					onclick={item.onclick}
					class="hover:cursor-pointer"
					disabled={item.disabled}
				>
					{#if item.icon}
						<Icon src={item.icon} />
					{/if}
					{item.label}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
