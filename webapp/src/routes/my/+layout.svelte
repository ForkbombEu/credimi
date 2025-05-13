<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BaseLayout from '$lib/layout/baseLayout.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import NavigationTabs from '@/components/ui-custom/navigationTabs.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';
	import { CheckCheck, GlobeIcon, Home, Shapes, TestTubeDiagonalIcon, User } from 'lucide-svelte';
	import type { Snippet } from 'svelte';

	//

	interface Props {
		children?: Snippet;
	}

	let { children }: Props = $props();
</script>

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
				{ title: m.Organization_page(), href: '/my/organization-page', icon: GlobeIcon },
				{ title: m.Profile(), href: '/my/profile', icon: User },
				{ title: m.Custom_checks(), href: '/my/custom-checks', icon: CheckCheck }
			]}
		/>
	</PageTop>

	<PageContent class="bg-secondary grow">
		{@render children?.()}
	</PageContent>
</BaseLayout>
