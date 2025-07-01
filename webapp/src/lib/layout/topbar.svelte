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

	function href(href: string) {
		return $featureFlags.DEMO ? '#waitlist' : href;
	}
</script>

<BaseTopbar class="bg-card border-none">
	{#snippet left()}
		<Button variant="link" href={href('/')}>
			<AppLogo />
		</Button>
		<div class="hidden flex-row sm:flex">
			<Button variant="link" href={href('/marketplace')}>
				{m.Marketplace()}
			</Button>
			<Button variant="link" href={href('/organizations')}>
				{m.organizations()}
			</Button>
			<Button variant="link" href="/news">{m.News()}</Button>
		</div>
	{/snippet}

	{#snippet right()}
		<div class="flex items-center space-x-2">
			<div class="hidden sm:flex sm:flex-row">
				<Button variant="link" href="/help">{m.Help()}</Button>
			</div>
			{#if !$featureFlags.DEMO && $featureFlags.AUTH}
				{#if !$currentUser}
					<Button variant="secondary" href="/login">{m.Login()}</Button>
				{:else}
					<Button variant="link" href="/my/tests/new">
						<Icon src={Sparkle} />
						{m.Start_a_new_check()}
					</Button>
					<Button variant="link" href="/my/services-and-products">
						<Icon src={LayoutDashboardIcon} />
						{m.Go_to_Dashboard()}
					</Button>
					<UserNav />
				{/if}
			{/if}
		</div>
	{/snippet}
</BaseTopbar>
