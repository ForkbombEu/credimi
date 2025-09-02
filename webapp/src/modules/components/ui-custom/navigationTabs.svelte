<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';

	import Icon from '@/components/ui-custom/icon.svelte';

	import NavigationTab, { type Props as NavigationTabProps } from './navigationTab.svelte';

	interface Props {
		tabs?: NavigationTabProps[];
	}

	let { tabs = [] }: Props = $props();

	// Find the active tab for mobile display
	const activeTab = $derived(
		tabs.find(
			(tab) => page.url.pathname === tab.href || page.url.pathname.startsWith(tab.href!)
		)
	);
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
			<h2 class="flex items-center gap-2 text-lg font-semibold text-foreground">
				{#if activeTab.icon}
					<Icon src={activeTab.icon} class="size-5" />
				{/if}
				{activeTab.title}
			</h2>
		</div>
	{/if}

	<ul class="scrollbar-hide flex gap-1 overflow-x-auto">
		{#each tabs as tab}
			<li role="presentation" class="flex-shrink-0">
				<NavigationTab {...tab} />
			</li>
		{/each}
	</ul>
</div>
