<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';

	import { page } from '$app/state';

	import type { LinkWithIcon } from '@/components/types';

	import * as Sidebar from '@/components/ui/sidebar';
	import { cn } from '@/components/ui/utils';
	import { localizeHref } from '@/i18n';

	import SidebarItemIcon from './sidebarItemIcon.svelte';

	type Props = HTMLAnchorAttributes &
		LinkWithIcon & {
			disabled?: boolean;
			sub?: boolean;
			isActive?: boolean;
			left?: Snippet;
		};

	const {
		href,
		title,
		icon,
		disabled = false,
		left,
		sub = false,
		onclick,
		...rest
	}: Props = $props();

	const Item = $derived(sub ? Sidebar.MenuSubItem : Sidebar.MenuItem);
	const Button = $derived(sub ? Sidebar.MenuSubButton : Sidebar.MenuButton);
	const isActive = $derived(page.url.pathname == href);

	const sidebarContext = Sidebar.useSidebar();
</script>

<Item class={cn({ 'cursor-not-allowed opacity-20': disabled })}>
	<Button {isActive}>
		{#snippet child({ props })}
			{#if !disabled}
				<a
					href={href ? localizeHref(href) : undefined}
					{...props}
					{...rest}
					onclick={(e) => {
						sidebarContext.setOpenMobile(false);
						onclick?.(e);
					}}
				>
					{@render content()}
				</a>
			{:else}
				<p {...props}>
					{@render content()}
				</p>
			{/if}
		{/snippet}
	</Button>
</Item>

{#snippet content()}
	{#if left}
		<SidebarItemIcon>
			{@render left()}
		</SidebarItemIcon>
	{:else if icon}
		<SidebarItemIcon src={icon} />
	{/if}

	<span>{title}</span>
{/snippet}
