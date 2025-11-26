<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import { slide } from 'svelte/transition';

	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Sidebar from '@/components/ui/sidebar/index.js';
	import { deLocalizeHref, localizeHref } from '@/i18n';

	import type { SidebarItem } from './sidebar';

	import SidebarItemComponent from './sidebar-item.svelte';

	//

	type Props = {
		item: SidebarItem;
	};

	let { item }: Props = $props();

	const isActive = $derived(deLocalizeHref(page.url.pathname) == item.url);
</script>

<Sidebar.MenuItem>
	<Sidebar.MenuButton {isActive}>
		{#snippet child({ props })}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={localizeHref(item.url)} {...props}>
				{#if item.component}
					<item.component title={item.title} />
				{:else}
					{#if item.icon}
						<Icon src={item.icon} />
					{/if}
					<span class="text-nowrap">
						{item.title}
					</span>
				{/if}
			</a>
		{/snippet}
	</Sidebar.MenuButton>
</Sidebar.MenuItem>

{#if isActive && item.children}
	<div transition:slide class="pl-4">
		{#each item.children as child (child.url)}
			<SidebarItemComponent item={child} />
		{/each}
	</div>
{/if}
