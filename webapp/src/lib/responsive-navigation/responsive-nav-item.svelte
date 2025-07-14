<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Component } from 'svelte';
	import type { Icon } from 'lucide-svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import IconComponent from '@/components/ui-custom/icon.svelte';
	import type { IconComponent as ComponentIcon } from '@/components/types';
	

	interface Props {
		href: string;
		label: string;
		icon?: ComponentIcon;
		variant?: 'desktop' | 'mobile';
		onClick?: () => void;
		onNavigate?: () => void;
		class?: string;
	}

	let { 
		href, 
		label, 
		icon, 
		variant = 'desktop', 
		onClick,
		onNavigate,
		class: className = '' 
	}: Props = $props();

	function handleClick() {
		onClick?.();
		// Close mobile menu when navigating (only for mobile variant)
		if (variant === 'mobile') {
			onNavigate?.();
		}
	}
</script>

{#if variant === 'desktop'}
	<Button variant="link" {href} onclick={handleClick} class={className}>
		{#if icon}
			<IconComponent src={icon} />
		{/if}
		{label}
	</Button>
{:else}
	<Button 
		variant="ghost" 
		{href} 
		onclick={handleClick}
		class="w-full justify-start text-left {className}"
	>
		{#if icon}
			<IconComponent src={icon} />
		{/if}
		{label}
	</Button>
{/if}
