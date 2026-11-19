<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { beforeNavigate } from '$app/navigation';
	import { page, updated } from '$app/state';
	import GlobalConfirm from '$lib/layout/global-confirm.svelte';
	import GlobalLoading from '$lib/layout/global-loading.svelte';
	import { queryClient } from '$lib/query-client';
	import SeoHead from '$lib/seo/seo-head.svelte';

	import { Toaster } from '@/components/ui/sonner';
	import { locales, localizeHref } from '@/i18n/paraglide/runtime';

	import './layout.css';

	let { children } = $props();

	// A tab opened before a deploy keeps running the old client, which can disagree with the
	// new API. Once version polling sees a new build, leave the old app at the next navigation.
	beforeNavigate(({ willUnload, to }) => {
		if (updated.current && !willUnload && to?.url) location.href = to.url.href;
	});
</script>

<SeoHead />

<QueryClientProvider client={queryClient}>
	{@render children()}

	<Toaster richColors closeButton class="dark!" />

	<GlobalLoading />

	<GlobalConfirm />

	<div style="display:none">
		{#each locales as locale (locale)}
			<a href={localizeHref(page.url.pathname, { locale })}>
				{locale}
			</a>
		{/each}
	</div>
</QueryClientProvider>
