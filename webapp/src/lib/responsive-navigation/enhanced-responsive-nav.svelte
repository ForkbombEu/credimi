<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ResponsiveNav } from './responsive-nav.svelte.js';
	import MobileNav from './mobile-nav.svelte';
	import NavItems, { type NavItem } from './nav-items.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		/**
		 * Navigation items configuration
		 */
		items?: NavItem[];
		/**
		 * Custom content for mobile menu (takes precedence over items)
		 */
		mobileContent?: Snippet;
		/**
		 * Custom content for desktop navigation (takes precedence over items)
		 */
		desktopContent?: Snippet;
		/**
		 * Navigation title for mobile menu
		 */
		mobileTitle?: string;
		/**
		 * Additional classes for the desktop container
		 */
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
<div class="hidden md:flex md:flex-row md:items-center md:space-x-1 min-w-0 overflow-hidden {className}">
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
			<NavItems {items} variant="mobile" />
		{/if}
	</MobileNav>
{/if}
