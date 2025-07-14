<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ResponsiveNav } from './responsive-nav.svelte.js';
	import MobileNav from './mobile-nav.svelte';
	import NavItems from './nav-items.svelte';
	import type { NavItem } from '@/components/types';
	import type { Snippet } from 'svelte';

	interface Props {
		items?: NavItem[];
		mobileContent?: Snippet;
		desktopContent?: Snippet;
		mobileTitle?: string;
		class?: string;
	}

	let { 
		items = [],
		mobileContent,
		desktopContent,
		mobileTitle = 'Navigation',
		class: className = '' 
	}: Props = $props();

	const nav = new ResponsiveNav();

	const hasMobileItems = $derived(
		mobileContent || items.some(item => 
			item.display === 'both' || 
			item.display === 'mobile-only' || 
			!item.display
		)
	);
</script>

<!-- Desktop Navigation -->
<div class="hidden lg:flex lg:flex-row lg:items-center lg:space-x-1 min-w-0 overflow-hidden {className}">
	{#if desktopContent}
		{@render desktopContent()}
	{:else}
		<NavItems {items} variant="desktop" />
	{/if}
</div>

<!-- Mobile Navigation Trigger & Sheet -->
{#if hasMobileItems}
	<MobileNav 
		open={nav.isOpen} 
		onOpenChange={nav.setOpen}
		title={mobileTitle}
	>
		{#if mobileContent}
			{@render mobileContent()}
		{:else}
			<NavItems {items} variant="mobile" onNavigate={nav.close} />
		{/if}
	</MobileNav>
{/if}
