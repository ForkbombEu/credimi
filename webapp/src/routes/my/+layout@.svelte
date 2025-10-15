<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- <script lang="ts">
	import BaseLayout from '$lib/layout/baseLayout.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { CheckCheck, GlobeIcon, Shapes, TestTubeDiagonalIcon, User } from 'lucide-svelte';

	import NavigationTabs from '@/components/ui-custom/navigationTabs.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	//

	let { children, data } = $props();
</script> -->
<!-- 
<BaseLayout>
	<PageTop contentClass="!pb-0">
		<T tag="h1">{m.hello_user({ username: $currentUser?.name! })}</T>

		<NavigationTabs
			tabs={[
				{
					title: m.Services_and_products(),
					href: '/my/services-and-products',
					icon: Shapes
				},
				{ title: m.Test_runs(), href: '/my/tests/runs', icon: TestTubeDiagonalIcon },
				{ title: m.Custom_checks(), href: '/my/custom-checks', icon: CheckCheck },
				{
					title: m.Organization(),
					href: '/my/organization',
					icon: GlobeIcon,
					notification: data.isOrganizationNotEdited
				},
				{ title: m.Profile(), href: '/my/profile', icon: User }
			]}
		/>
	</PageTop>

	<PageContent class="grow bg-secondary">
		{@render children?.()}
	</PageContent>
</BaseLayout> -->

<script lang="ts">
	import { Separator } from '@/components/ui/separator/index.js';
	import * as Sidebar from '@/components/ui/sidebar/index.js';

	import { data } from './_partials/sidebar-data';
	import AppSidebar from './_partials/sidebar.svelte';

	let { children } = $props();
</script>

<Sidebar.Provider>
	<AppSidebar {data} />
	<Sidebar.Inset>
		<header class="flex shrink-0 items-center gap-2 border-b px-4">
			<Sidebar.Trigger class="-ml-1" />
			<Separator orientation="vertical" class="mr-2 h-4" />
		</header>
		<div class="flex flex-1 flex-col gap-6 p-8">
			{@render children?.()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
