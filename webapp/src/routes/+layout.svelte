<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { appName } from '@/brand';
	import { Toaster } from '@/components/ui/sonner';
	import { locales, localizeHref } from '@/i18n';

	let { children } = $props();
</script>

<svelte:head>
	<title>
		{appName}
	</title>
</svelte:head>

<!-- Needed for site crawling for prerendering -->
<!-- https://inlang.com/m/gerre34r/library-inlang-paraglideJs/server-side-rendering#site-crawling-invisible-anchor-tags -->
<div style="display: none">
	{#each locales as locale}
		<a href={localizeHref(page.url.pathname, { locale })}>{locale}</a>
	{/each}
</div>

{@render children()}

<Toaster richColors closeButton />
