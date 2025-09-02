<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { URL_SEARCH_PARAM_NAME } from '$lib/content';
	import { getTagTranslation } from '$lib/content/tags-i18n';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import TableOfContents from 'lucide-svelte/icons/table-of-contents';
	import Toc from 'svelte-toc';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Popover from '@/components/ui/popover';
	import { m } from '@/i18n';

	import Breadcrumbs from './slug-breadcrumbs.svelte';

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
					<div class="flex gap-1 text-muted-foreground">
						<T tag="small" class="text-balance !font-normal">Published on:</T>
						<T tag="small" class="text-balance !font-semibold">
							{date.toLocaleDateString()}
						</T>
					</div>
				{/if}

				<T tag="h1" class="text-balance !font-bold">{title}</T>
			</div>

			<T tag="p" class="text-balance !font-bold text-primary">
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
					<Toc {headingSelector} minItems={1} title="" breakpoint={100} />
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

	:global(.toc-sidebar aside.toc > nav > ol > li.active),
	:global(.toc-mobile aside.toc > nav > ol > li.active) {
		font-weight: 600;
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
