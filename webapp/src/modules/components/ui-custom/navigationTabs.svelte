<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import Icon from '@/components/ui-custom/icon.svelte';
	import type { LinkWithIcon } from '../types';
	import NavigationTab, { type Props as NavigationTabProps } from './navigationTab.svelte';

	interface Props {
		tabs?: NavigationTabProps[];
	}

	let { tabs = [] }: Props = $props();

	// Find the active tab for mobile display
	const activeTab = $derived(tabs.find(tab => page.url.pathname === tab.href));
</script>

<!-- Desktop view: Normal flex layout -->
<div class="hidden lg:block">
	<ul class="flex gap-1">
		{#each tabs as tab}
			<li role="presentation">
				<NavigationTab {...tab} />
			</li>
		{/each}
	</ul>
</div>

<!-- Mobile view: Active tab title + horizontal scrolling tabs -->
<div class="block lg:hidden">
	<!-- Active tab title for mobile -->
	{#if activeTab}
		<div class="mb-3 px-1">
			<h2 class="text-lg font-semibold text-foreground flex items-center gap-2">
				{#if activeTab.icon}
					<Icon src={activeTab.icon} class="size-5" />
				{/if}
				{activeTab.title}
			</h2>
		</div>
	{/if}
	
	<!-- Icon-only tabs -->
	<ul class="flex gap-1 overflow-x-auto scrollbar-hide pb-1">
		{#each tabs as tab}
			<li role="presentation" class="flex-shrink-0">
				<NavigationTab {...tab} />
			</li>
		{/each}
	</ul>
</div>
