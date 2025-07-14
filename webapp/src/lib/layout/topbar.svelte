<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Button from '@/components/ui-custom/button.svelte';
	import { featureFlags } from '@/features';
	import { m } from '@/i18n';
	import BaseTopbar from '@/components/layout/topbar.svelte';
	import { currentUser } from '@/pocketbase';
	import UserNav from './userNav.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { LayoutDashboardIcon, Sparkle } from 'lucide-svelte';
	import { AppLogo } from '@/brand';
	import { Badge } from '@/components/ui/badge';
	import ResponsiveNav from '../responsive-navigation/responsive-nav.svelte';
	import type { NavItem } from '@/components/types';

	function href(href: string) {
		return $featureFlags.DEMO ? '#waitlist' : href;
	}

	const navigationItems: NavItem[] = [
		{
			href: href('/marketplace'),
			label: m.Marketplace(),
			display: 'both'
		},
		{
			href: href('/organizations'),
			label: m.organizations(),
			display: 'both'
		},
		{
			href: '/news',
			label: m.News(),
			display: 'both'
		},
		{
			href: 'https://docs.credimi.io',
			label: m.Help(),
			display: 'mobile-only'
		}
	];

	// Add authenticated user items that only appear in mobile
	const userItems: NavItem[] = $derived(
		!$featureFlags.DEMO && $featureFlags.AUTH && $currentUser ? [
			{
				href: '/my/tests/new',
				label: m.Start_a_new_check(),
				icon: Sparkle,
				display: 'mobile-only'
			},
			{
				href: '/my/services-and-products',
				label: m.Go_to_Dashboard(),
				icon: LayoutDashboardIcon,
				display: 'mobile-only'
			}
		] : []
	);

	const allItems = $derived([...navigationItems, ...userItems]);
</script>

<BaseTopbar class="bg-card border-none">
	{#snippet left()}
		<div class="flex items-center space-x-4 min-w-0 overflow-hidden">
			<Button variant="link" href={href('/')} class="shrink-0">
				<AppLogo />
			</Button>
			
			<!-- Desktop navigation only -->
			<div class="hidden md:flex md:flex-row md:items-center md:space-x-1 min-w-0 overflow-hidden">
				<Button variant="link" href={href('/marketplace')}>
					{m.Marketplace()}
				</Button>
				<Button variant="link" href={href('/organizations')}>
					{m.organizations()}
				</Button>
				<Button variant="link" href="/news">
					{m.News()}
				</Button>
			</div>
		</div>
	{/snippet}

	{#snippet right()}
		<div class="flex items-center space-x-2 min-w-0 overflow-hidden">
			<!-- Help link only appears on desktop (mobile has it in the menu) -->
			<div class="hidden sm:flex sm:flex-row">
				<Button variant="link" href="https://docs.credimi.io">{m.Help()}</Button>
			</div>
			
			{#if !$featureFlags.DEMO && $featureFlags.AUTH}
				{#if !$currentUser}
					<Button variant="secondary" href="/login">{m.Login()}</Button>
					
					<!-- Mobile menu trigger - only show when not logged in -->
					<div class="md:hidden">
						<ResponsiveNav 
							items={navigationItems}
							mobileTitle="Navigation"
						/>
					</div>
				{:else}
					<!-- User action buttons only on desktop (mobile has them in menu) -->
					<div class="hidden sm:flex sm:flex-row sm:items-center sm:space-x-2 min-w-0 overflow-hidden">
						<Button variant="link" href="/my/tests/new" class="text-nowrap">
							<Icon src={Sparkle} />
							{m.Start_a_new_check()}
							<Badge
								variant="outline"
								class="border-primary text-primary !hover:no-underline text-xs ml-2"
							>
								{m.Beta()}
							</Badge>
						</Button>
						<Button variant="link" href="/my/services-and-products" class="text-nowrap">
							<Icon src={LayoutDashboardIcon} />
							{m.Go_to_Dashboard()}
						</Button>
					</div>
					
					<UserNav />
					
					<div class="md:hidden">
						<ResponsiveNav 
							items={allItems}
							mobileTitle="Navigation"
						/>
					</div>
				{/if}
			{:else}
				<div class="md:hidden">
					<ResponsiveNav 
						items={navigationItems}
						mobileTitle="Navigation"
					/>
				</div>
			{/if}
		</div>
	{/snippet}
</BaseTopbar>
