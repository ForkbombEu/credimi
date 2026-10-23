<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ContentSeo } from './seo';

	type Props = { seo: ContentSeo; tags: string[] };

	const { seo, tags }: Props = $props();
</script>

<svelte:head>
	<title>{seo.title}</title>
	<meta name="description" content={seo.description} />
	<meta name="keywords" content={seo.keywords} />
	<link rel="canonical" href={seo.canonicalUrl} />

	<meta property="og:type" content="article" />
	<meta property="og:site_name" content="Credimi" />
	<meta property="og:title" content={seo.title} />
	<meta property="og:description" content={seo.description} />
	<meta property="og:url" content={seo.canonicalUrl} />
	<meta property="og:image" content={seo.socialImageUrl} />
	<meta property="article:published_time" content={seo.publishedTime} />
	<meta property="article:modified_time" content={seo.modifiedTime} />

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={seo.title} />
	<meta name="twitter:description" content={seo.description} />
	<meta name="twitter:image" content={seo.socialImageUrl} />

	{#each tags as tag (tag)}
		<meta property="article:tag" content={tag} />
	{/each}
</svelte:head>
