<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentType } from 'svelte';
	import type { Icon } from 'lucide-svelte';
	import ResponsiveNavItem from './responsive-nav-item.svelte';

	export interface NavItem {
		href: string;
		label: string;
		icon?: ComponentType<Icon>;
		/**
		 * Controls where this item appears:
		 * - 'both': appears in both desktop and mobile (default)
		 * - 'desktop-only': only appears in desktop navigation
		 * - 'mobile-only': only appears in mobile menu
		 */
		display?: 'both' | 'desktop-only' | 'mobile-only';
		/**
		 * Custom click handler
		 */
		onClick?: () => void;
		/**
		 * Additional CSS classes
		 */
		class?: string;
	}

	interface Props {
		items: NavItem[];
		variant: 'desktop' | 'mobile';
	}

	let { items, variant }: Props = $props();

	const filteredItems = $derived(
		items.filter(item => {
			if (item.display === 'both' || !item.display) return true;
			if (variant === 'desktop' && item.display === 'desktop-only') return true;
			if (variant === 'mobile' && item.display === 'mobile-only') return true;
			return false;
		})
	);
</script>

{#each filteredItems as item}
	<ResponsiveNavItem 
		href={item.href}
		label={item.label}
		icon={item.icon}
		{variant}
		onClick={item.onClick}
		class={item.class}
	/>
{/each}
