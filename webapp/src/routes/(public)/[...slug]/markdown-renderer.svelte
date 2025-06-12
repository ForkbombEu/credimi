<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageTop from '$lib/layout/pageTop.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import Breadcrumbs from './slug-breadcrumbs.svelte';
	import { marked } from 'marked';
	import T from '@/components/ui-custom/t.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import fm from 'front-matter';
	import { pageFrontMatterSchema } from '$lib/content';

	type Props = {
		content: string;
		class?: string;
		preprocessHtml?: (html: string) => string;
	};

	let { content, class: className = '', preprocessHtml }: Props = $props();
	const { attributes, body } = $derived(fm<{ [key: string]: any }>(content));
	let parsedFrontMatter = $derived.by(() => {
		const parsedFrontMatter = pageFrontMatterSchema.safeParse(attributes);
		return parsedFrontMatter.data;
	});

	const html = $derived.by(() => {
		const baseHtml = marked(body, { async: false });
		return preprocessHtml ? preprocessHtml(baseHtml) : baseHtml;
	});
</script>

<PageTop containerClass="border-t-0" contentClass={'pt-4'}>
	<Breadcrumbs />
	{#if parsedFrontMatter}
		{@const { title, description, tags, date } = parsedFrontMatter}
		<div class="mx-auto max-w-screen-lg">
			<div class="space-y-2">
				<T tag="h1" class="text-balance !font-bold">
					{title}
				</T>
				<div class="flex flex-col gap-2 py-2">
					<T tag="p" class="text-primary text-balance !font-bold">
						{description}
					</T>
				</div>
			</div>
			{#if date}
				<div class="my-7 flex gap-2">
					<T tag="small" class="text-balance !font-normal">
						{'Published on:'}
					</T>
					<T tag="small" class="text-balance !font-bold">
						{date.toLocaleDateString()}
					</T>
				</div>
			{/if}
			{#if tags.length}
				<div class="!mb-1">
					<T tag="small" class="text-balance !font-normal">
						{'Tags:'}
					</T>
				</div>
				<div class="!mt-0 flex items-center gap-4">
					{#each tags as tag}
						<Button
							variant="outline"
							class="border-primary text-primary"
							href={`tags?search=${tag}`}
						>
							{tag}
						</Button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</PageTop>

<div class="bg-secondary">
	<PageContent>
		<div class="mx-auto max-w-screen-lg">
			<div class="prose prose-headings:font-serif prose-h1:text-3xl {className}">
				{@html html}
			</div>
		</div>
	</PageContent>
</div>
