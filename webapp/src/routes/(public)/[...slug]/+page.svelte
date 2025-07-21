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
				<div class="flex items-center gap-2 flex-wrap">
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
		<div class="hidden lg:block w-64 flex-shrink-0">
			<div class="toc-sidebar sticky top-5">
				<Toc {headingSelector} minItems={1} title="">
					<!-- @ts-ignore -->
					<span slot="toc-item" let:heading class="flex items-center gap-2">
						<svg
							class="size-4 shrink-0"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path
								d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"
							/>
							<polyline points="14,2 14,8 20,8" />
						</svg>
						{heading.textContent}
					</span>
				</Toc>
			</div>
		</div>
		
		<!-- Main Content -->
		<div class="flex-1 mx-auto max-w-screen-lg">
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
