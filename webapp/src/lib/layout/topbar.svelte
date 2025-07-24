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
	import MobileNav from './nav-mobile.svelte';
	import type { LinkWithIcon } from '@/components/types';
	import { fromStore } from 'svelte/store';
	import NavLink from './nav-link.svelte';

	//

	const featureFlagsState = fromStore(featureFlags);
	const currentUserState = fromStore(currentUser);

	function href(href: string) {
		return featureFlagsState.current.DEMO ? '#waitlist' : href;
	}

	const leftItems: LinkWithIcon[] = [
		{
			href: href('/marketplace'),
			title: m.Marketplace()
		},
		{
			href: href('/organizations'),
			title: m.organizations()
		},
		{
			href: '/news',
			title: m.News()
		}
	];

	const rightItems: LinkWithIcon[] = $derived.by(() => {
		const { DEMO, AUTH } = featureFlagsState.current;
		const user = currentUserState.current;

		const items: LinkWithIcon[] = [
			{
				href: 'https://docs.credimi.io',
				title: m.Help(),
				target: '_blank'
			}
		];

		if (!DEMO && AUTH && user) {
			items.push(
				{
					href: '/my/tests/new',
					title: m.Start_a_new_check(),
					icon: Sparkle
				},
				{
					href: '/my/services-and-products',
					title: m.Go_to_Dashboard(),
					icon: LayoutDashboardIcon
				}
			);
		}

		return items;
	});

	const allItems = $derived([...leftItems, ...rightItems]);
</script>

<BaseTopbar class="bg-card border-none">
	{#snippet left()}
		<div class="flex min-w-0 items-center space-x-4 overflow-hidden">
			<Button variant="link" href={href('/')} class="shrink-0">
				<AppLogo />
			</Button>

			<div class="hidden lg:flex lg:flex-row lg:items-center lg:gap-1">
				{#each leftItems as item}
					<NavLink link={item} variant="desktop" />
				{/each}
			</div>
		</div>
	{/snippet}

	{#snippet right()}
		<div class="flex items-center gap-2">
			<div class="hidden lg:flex lg:flex-row">
				{#each rightItems as item}
					<NavLink
						link={item}
						variant="desktop"
						badge={item.href?.endsWith('/my/tests/new') ? m.Beta() : undefined}
					/>
				{/each}
			</div>

			<UserNav />

			<div class="lg:hidden">
				<MobileNav items={allItems} />
			</div>
		</div>
	{/snippet}
</BaseTopbar>
