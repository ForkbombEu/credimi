<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageTop from '$lib/layout/pageTop.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import Breadcrumbs from './slug-breadcrumbs.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { URL_SEARCH_PARAM_NAME } from '$lib/content';
	import { getTagTranslation } from '$lib/content/tags-i18n';
	import Toc from 'svelte-toc';

	const { data } = $props();
	let { attributes, body } = data;

	const headingSelector =
		'#content-area h1, #content-area h2, #content-area h3, #content-area h4, #content-area h5, #content-area h6';
</script>

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
					{#each tags as tag}
						<Button
							size="sm"
							variant="outline"
							class="border-primary text-primary"
							href={`tags?${URL_SEARCH_PARAM_NAME}=${tag}`}
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
	<div class="flex gap-8">
		<div class="hidden w-64 flex-shrink-0 lg:block">
			<div class="toc-sidebar sticky top-5">
				<Toc {headingSelector} minItems={1} title="" />
			</div>
		</div>

		<!-- Main Content -->
		<div class="mx-auto max-w-screen-lg flex-1">
			<div class="prose prose-h1:text-3xl" id="content-area">
				{@html body}
			</div>
		</div>
	</div>
</PageContent>

<style>
	/* Remove spacing between TOC items */
	:global(.toc-sidebar aside.toc > nav > ol > li) {
		margin-bottom: 0;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li:last-child) {
		margin-bottom: 0;
	}

	/* Use proper svelte-toc CSS custom properties */
	:global(.toc-sidebar) {
		/* Override default padding to remove left and top padding */
		--toc-padding: 0 1em 0 0;

		/* Match paragraph font size */
		font-size: 1rem;
		line-height: 1.75;

		/* Ensure text and icons are visible */
		color: hsl(var(--foreground));

		/* Active state styling */
		--toc-active-bg: transparent;
		--toc-active-color: hsl(var(--primary));
		--toc-active-li-font: 600;
		--toc-active-border: none;
		--toc-active-border-radius: 0;

		/* Hover state styling */
		--toc-li-hover-color: inherit;
		--toc-li-hover-bg: transparent;

		/* General list item styling */
		--toc-li-padding: 0;
		--toc-li-margin: 0;
		--toc-li-border: none;
		--toc-li-border-radius: 0;

		/* Remove default list styling */
		--toc-ol-list-style: none;
		--toc-ol-padding: 0;
		--toc-ol-margin: 0;
	}

	:global(.toc-sidebar aside.toc > nav > ol > li.active) {
		font-weight: 600;
	}
</style>
