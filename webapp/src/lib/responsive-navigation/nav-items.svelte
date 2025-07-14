<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ResponsiveNavItem from './responsive-nav-item.svelte';
	import type { NavItem } from '@/components/types';
	
	interface Props {
		items: NavItem[];
		variant: 'desktop' | 'mobile';
		onNavigate?: () => void;
	}

	const { items, variant, onNavigate }: Props = $props();

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
		{onNavigate}
		class={item.class}
	/>
{/each}
