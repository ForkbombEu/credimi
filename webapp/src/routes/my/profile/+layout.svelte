<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import NavSidebar from '@/components/ui-custom/navSidebar.svelte';
	import type { LinkWithIcon } from '@/components/types';
	import { KeyIcon, LockIcon, UserIcon } from 'lucide-svelte';
	import { fromStore } from 'svelte/store';
	import { featureFlags } from '@/features';
	import { PageCard } from '@/components/layout';

	//

	const { children } = $props();

	const links = $state<LinkWithIcon[]>([
		{
			href: '/my/profile',
			title: 'Personal profile',
			icon: UserIcon
		},
		{
			href: '/my/profile/webauthn',
			title: 'Webauthn',
			icon: LockIcon
		}
	]);

	const featuresState = fromStore(featureFlags);
	$effect(() => {
		if (featuresState.current.KEYPAIROOM) {
			links.push({
				href: '/my/profile/public-keys',
				title: 'Public Keys',
				icon: KeyIcon
			});
		}
	});
</script>

<div class="flex w-full flex-col gap-8 sm:flex-row">
	<NavSidebar title="settings" {links}></NavSidebar>
	<div class="grow">
		<PageCard>
			{@render children()}
		</PageCard>
	</div>
</div>
