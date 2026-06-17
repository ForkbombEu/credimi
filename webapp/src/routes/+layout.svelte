<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import favicon from '$lib/assets/favicon.png';
	import GlobalConfirm from '$lib/layout/global-confirm.svelte';
	import GlobalLoading from '$lib/layout/global-loading.svelte';

	import { appName } from '@/brand';
	import { Toaster } from '@/components/ui/sonner';
	import { locales, localizeHref } from '@/i18n/paraglide/runtime';

	import './layout.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>
		{appName}
	</title>
</svelte:head>

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
