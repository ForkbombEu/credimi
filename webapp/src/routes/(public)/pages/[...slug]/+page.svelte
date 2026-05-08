<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { URL_SEARCH_PARAM_NAME } from '$lib/content';
	import { getTagTranslation } from '$lib/content/tags-i18n';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import TableOfContents from '@lucide/svelte/icons/table-of-contents';
	import Toc from 'svelte-toc';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Popover from '@/components/ui/popover';
	import { m } from '@/i18n';

	import Breadcrumbs from './slug-breadcrumbs.svelte';

	const { data } = $props();
	let { attributes, body } = data;
	let activeHeading = $state<HTMLHeadingElement | null>(null);

	const headingSelector =
		'#content-area h1, #content-area h2, #content-area h3, #content-area h4, #content-area h5, #content-area h6';

	function getTocLabel(heading: HTMLHeadingElement) {
		return heading.textContent?.trim() ?? '';
	}
</script>

<svelte:head>
	<title>{data.seo.title}</title>
	<meta name="description" content={data.seo.description} />
	<meta name="keywords" content={data.seo.keywords} />
	<link rel="canonical" href={data.seo.canonicalUrl} />

	<meta property="og:type" content="article" />
	<meta property="og:site_name" content="Credimi" />
	<meta property="og:title" content={data.seo.title} />
	<meta property="og:description" content={data.seo.description} />
	<meta property="og:url" content={data.seo.canonicalUrl} />
	<meta property="og:image" content={data.seo.socialImageUrl} />
	<meta property="article:published_time" content={data.seo.publishedTime} />
	<meta property="article:modified_time" content={data.seo.modifiedTime} />

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={data.seo.title} />
	<meta name="twitter:description" content={data.seo.description} />
	<meta name="twitter:image" content={data.seo.socialImageUrl} />

	{#each attributes.tags as tag (tag)}
		<meta property="article:tag" content={tag} />
	{/each}
</svelte:head>

<PageTop containerClass="border-t-0" contentClass="pt-8 !space-y-12">
	<Breadcrumbs />

	{#if attributes}
		{@const { title, description, tags, date } = attributes}
		<div class="max-w-prose space-y-6">
			<div class="space-y-2">
				{#if date}
					<div class="text-muted-foreground flex gap-1">
						<T tag="small" class="text-balance !font-normal">Published on:</T>
						<T tag="small" class="text-balance !font-semibold">
							{date.toLocaleDateString()}
						</T>
					</div>
				{/if}

				<T tag="h1" class="text-balance !font-bold">{title}</T>
			</div>

			<T tag="p" class="text-primary text-balance !font-bold">
				{description}
			</T>

			{#if tags.length}
				<div class="flex flex-wrap items-center gap-2">
					{#each tags as tag (tag)}
						<Button
							size="sm"
							variant="outline"
							class="border-primary text-primary"
							href={`/pages/tags?${URL_SEARCH_PARAM_NAME}=${tag}`}
						>
							{getTagTranslation(tag)}
						</Button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</PageTop>

<PageContent class="bg-secondary">
	<div class="flex gap-8 xl:gap-12">
		<div class="hidden w-72 flex-shrink-0 lg:block">
			<div class="toc-sidebar sticky top-5 rounded-2xl border border-border/70 bg-background/80 p-5 shadow-sm backdrop-blur-sm">
				<div class="mb-4 border-b border-border/70 pb-3">
					<T tag="p" class="text-muted-foreground text-sm font-medium">
						{m.toc()}
					</T>
				</div>
				<Toc bind:activeHeading {headingSelector} minItems={1} title="">
					{#snippet tocItem(heading: HTMLHeadingElement)}
						<span
							class:toc-entry-active={heading === activeHeading}
							class="toc-entry"
						>
							{getTocLabel(heading)}
						</span>
					{/snippet}
				</Toc>
			</div>
		</div>

		<!-- Main Content -->
		<div class="mx-auto min-w-0 max-w-4xl flex-1">
			<div
				class="prose prose-sm prose-headings:scroll-mt-24 prose-headings:font-semibold prose-p:text-foreground/90 prose-li:text-foreground/90 prose-a:text-primary prose-a:font-medium prose-a:no-underline hover:prose-a:underline prose-pre:rounded-xl prose-pre:border prose-pre:border-border/70 prose-blockquote:border-l-primary prose-blockquote:text-foreground/80 prose-table:w-full prose-table:table-fixed prose-th:border-b prose-th:border-border/70 prose-th:px-3 prose-th:py-2 prose-th:text-left prose-th:text-[0.72rem] prose-th:font-semibold prose-th:tracking-[0.08em] prose-th:text-foreground prose-th:uppercase prose-td:border-b prose-td:border-border/50 prose-td:px-3 prose-td:py-2 prose-td:align-top prose-td:text-[0.92em] sm:prose-base lg:prose-lg prose-h1:text-3xl prose-h2:text-2xl"
				id="content-area"
			>
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html body}
			</div>
		</div>
	</div>
</PageContent>

<div class="fixed bottom-6 right-6 z-50 lg:hidden">
	<Popover.Root>
		<Popover.Trigger>
			{#snippet child({ props })}
				<Button
					{...props}
					class="flex h-12 items-center gap-2 rounded-full px-4 shadow-lg transition-shadow hover:shadow-xl"
				>
					<TableOfContents class="h-4 w-4" />
					<span class="text-sm font-medium">{m.toc()}</span>
				</Button>
			{/snippet}
		</Popover.Trigger>

		<Popover.Content side="top" align="end" class="w-64">
			<div class="space-y-2">
				<div class="toc-mobile">
					<Toc bind:activeHeading {headingSelector} minItems={1} title="" breakpoint={100}>
						{#snippet tocItem(heading: HTMLHeadingElement)}
							<span
								class:toc-entry-active={heading === activeHeading}
								class="toc-entry"
							>
								{getTocLabel(heading)}
							</span>
						{/snippet}
					</Toc>
				</div>
			</div>
		</Popover.Content>
	</Popover.Root>
</div>

<style>
	/* Remove spacing between TOC items */
	:global(.toc-sidebar aside.toc > nav > ol > li) {
		margin-bottom: 0;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li:last-child) {
		margin-bottom: 0;
	}

	/* Use proper svelte-toc CSS custom properties */
	:global(.toc-sidebar),
	:global(.toc-mobile) {
		/* Override default padding to remove left and top padding */
		--toc-padding: 0;

		/* Match paragraph font size */
		font-size: 0.9rem;
		line-height: 1.55;

		/* Ensure text and icons are visible */
		color: hsl(var(--foreground));

		/* Active state styling */
		--toc-active-bg: transparent;
		--toc-active-color: inherit;
		--toc-active-li-font: initial;
		--toc-active-border: none;
		--toc-active-border-radius: 0.65rem;

		/* Hover state styling */
		--toc-li-hover-color: hsl(var(--foreground));
		--toc-li-hover-bg: transparent;

		/* General list item styling */
		--toc-li-padding: 0.2rem 0;
		--toc-li-margin: 0;
		--toc-li-border: none;
		--toc-li-border-radius: 0;

		/* Remove default list styling */
		--toc-ol-list-style: none;
		--toc-ol-padding: 0;
		--toc-ol-margin: 0;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li.active),
	:global(.toc-mobile aside.toc > nav > ol > li.active) {
		background: transparent !important;
		border: none !important;
		border-radius: 0 !important;
		color: inherit !important;
		font-weight: 400 !important;
		box-shadow: none !important;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li),
	:global(.toc-mobile aside.toc > nav > ol > li) {
		color: hsl(var(--muted-foreground));
		padding: 0.12rem 0;
		transition:
			color 150ms ease,
			background-color 150ms ease,
			border-color 150ms ease,
			box-shadow 150ms ease;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li:hover),
	:global(.toc-mobile aside.toc > nav > ol > li:hover) {
		color: hsl(var(--foreground));
	}

	:global(.toc-entry) {
		display: block;
		padding: 0.38rem 0.65rem;
		font-size: 0.92rem;
		font-weight: 400;
		letter-spacing: -0.01em;
		border: 1px solid transparent;
		border-radius: 0.65rem;
		color: inherit;
		background: transparent;
		transition:
			background-color 150ms ease,
			border-color 150ms ease,
			box-shadow 150ms ease;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li:hover .toc-entry),
	:global(.toc-mobile aside.toc > nav > ol > li:hover .toc-entry) {
		background: color-mix(in oklab, hsl(var(--accent)) 60%, transparent);
	}

	:global(.toc-entry-active) {
		background: oklch(0.9464 0.0284 294.59) !important;
		border-color: transparent !important;
		box-shadow: inset 3px 0 0 hsl(var(--primary)) !important;
	}

	/* Mobile ToC specific styles */
	:global(.toc-mobile) {
		--toc-padding: 0;
		font-size: 0.875rem;
		line-height: 1.5;
	}

	:global(.toc-mobile aside.toc > nav > ol > li) {
		margin-bottom: 0.25rem;
	}

	:global(.toc-mobile aside.toc > nav > ol > li:last-child) {
		margin-bottom: 0;
	}
</style>
