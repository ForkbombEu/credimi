<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { type Snippet } from 'svelte';

	type DashboardNavbar = {
		title: string | undefined;
		right: (() => ReturnType<Snippet>) | undefined;
	};

	export const dashboardNavbar: DashboardNavbar = $state({
		title: 'Dashboard',
		right: undefined
	});

	export function setDashboardNavbar(navbar: Partial<DashboardNavbar>) {
		$effect(() => {
			if (page.url) {
				dashboardNavbar.title = navbar.title;
				dashboardNavbar.right = navbar.right;
			}
		});
	}
</script>

<script lang="ts">
	import { page } from '$app/state';

	import { Separator } from '@/components/ui/separator/index.js';
	import * as Sidebar from '@/components/ui/sidebar/index.js';

	import { getSidebarData } from './_partials/sidebar-data.svelte';
	import AppSidebar from './_partials/sidebar.svelte';

	let { children } = $props();

	// onNavigate(() => {
	// 	dashboardNavbar.title = undefined;
	// 	dashboardNavbar.right = undefined;
	// });
</script>

<Sidebar.Provider>
	<AppSidebar data={getSidebarData()} />
	<Sidebar.Inset class="overflow-x-hidden">
		<header class="sticky top-0 flex h-14 shrink-0 border-b bg-background">
			<div class="flex items-center">
				<div class="p-2">
					<Sidebar.Trigger />
				</div>
				<Separator orientation="vertical" class="mr-2 h-4" />
			</div>
			<div class="flex grow items-center justify-between gap-2 py-2 pr-4 pl-2">
				<p class="font-semibold">{dashboardNavbar.title}</p>
				{@render dashboardNavbar.right?.()}
			</div>
		</header>
		<div class="flex w-full flex-1 flex-col gap-6 overflow-x-hidden p-4">
			{@render children?.()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
