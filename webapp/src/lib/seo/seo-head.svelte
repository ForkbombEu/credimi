<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';

	import { appName } from '@/brand';
	import { getLocale } from '@/i18n';

	import {
		absoluteUrl,
		canonicalUrl,
		isIndexableRoute,
		jsonLdScriptTag,
		localeAlternates,
		ogImages,
		organizationJsonLd,
		SITE_DESCRIPTION,
		SITE_ORIGIN,
		websiteJsonLd,
		type PageSeo
	} from './index';

	// The single place that writes SEO tags: routes only declare `seo` in their load data.
	const seo: PageSeo = $derived(page.data.seo ?? {});
	const origin = SITE_ORIGIN;

	const title = $derived(seo.title ? `${seo.title} | ${appName}` : appName);
	const description = $derived(seo.description ?? SITE_DESCRIPTION);
	const image = $derived(absoluteUrl(seo.image ?? ogImages.default, origin));
	const imageAlt = $derived(seo.imageAlt ?? seo.title ?? appName);
	const canonical = $derived(canonicalUrl(page.url.pathname, origin, getLocale(), seo.locales));
	const indexable = $derived(
		!seo.noindex && page.status < 400 && isIndexableRoute(page.route.id)
	);
	const alternates = $derived(
		indexable ? localeAlternates(page.url.pathname, origin, seo.locales) : []
	);
	const jsonLd = $derived(
		indexable ? [organizationJsonLd(origin), websiteJsonLd(origin), ...(seo.jsonLd ?? [])] : []
	);
</script>

<svelte:head>
	<title>{title}</title>
	<meta name="description" content={description} />
	{#if indexable}
		<meta name="robots" content="index, follow, max-image-preview:large, max-snippet:-1" />
		<link rel="canonical" href={canonical} />
		{#each alternates as alternate (alternate.hreflang)}
			<link rel="alternate" hreflang={alternate.hreflang} href={alternate.href} />
		{/each}
	{:else}
		<meta name="robots" content="noindex, follow" />
	{/if}

	<meta property="og:site_name" content={appName} />
	<meta property="og:type" content={seo.type ?? 'website'} />
	<meta property="og:title" content={seo.title ?? appName} />
	<meta property="og:description" content={description} />
	<meta property="og:url" content={canonical} />
	<meta property="og:image" content={image} />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta property="og:image:alt" content={imageAlt} />
	<meta property="og:locale" content={getLocale()} />
	{#if seo.article}
		<meta property="article:published_time" content={seo.article.publishedTime} />
		<meta property="article:modified_time" content={seo.article.modifiedTime} />
		{#each seo.article.tags as tag (tag)}
			<meta property="article:tag" content={tag} />
		{/each}
	{/if}

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={seo.title ?? appName} />
	<meta name="twitter:description" content={description} />
	<meta name="twitter:image" content={image} />
	<meta name="twitter:image:alt" content={imageAlt} />

	{#each jsonLd as data, i (i)}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -- serialized JSON with `<` escaped (jsonLdScriptTag) -->
		{@html jsonLdScriptTag(data)}
	{/each}
</svelte:head>
