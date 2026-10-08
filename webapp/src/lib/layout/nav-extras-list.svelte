<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MouseEventHandler } from 'svelte/elements';

	import Separator from '@/components/ui/separator/separator.svelte';

	import NavExtraLink from './nav-extra-link.svelte';
	import { extraGroups } from './topbar-links';

	interface Props {
		onclick?: MouseEventHandler<HTMLAnchorElement>;
	}

	const { onclick }: Props = $props();
</script>

<div class="cursor-pointer">
	{#each extraGroups as group, index (group.label)}
		{#if index > 0}
			<Separator />
		{/if}
		<div class={['py-1', index == extraGroups.length - 1 && 'pb-0']}>
			<p
				class="cursor-default px-3 pt-2 pb-1 text-[10px] font-medium tracking-wide text-muted-foreground uppercase"
			>
				{group.label}
			</p>
			{#each group.items as item (item.href)}
				<NavExtraLink link={item} {onclick} />
			{/each}
		</div>
	{/each}
</div>
