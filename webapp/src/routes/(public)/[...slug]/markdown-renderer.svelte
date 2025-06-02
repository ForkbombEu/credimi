<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageTop from '$lib/layout/pageTop.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';

	import { marked } from 'marked';
	import T from '@/components/ui-custom/t.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Breadcrumb from '@/components/ui/breadcrumb/breadcrumb.svelte';
	import Breadcrumbs from '@/components/ui-custom/breadcrumbs.svelte';

	type Props = {
		content: string;
		class?: string;
		preprocessHtml?: (html: string) => string;
	};

	let { content, class: className = '', preprocessHtml }: Props = $props();

	const fmRegex = /^---\s*[\r\n]+([\s\S]*?)^---\s*[\r\n]+/m;
	const fmMatch = content.match(fmRegex);

	let rawMeta = '';
	let body = content;
	if (fmMatch) {
		rawMeta = fmMatch[1];
		body = content.slice(fmMatch[0].length);
	}

	const tags: string[] = [];
	let publishedDate = '';
	let title = '';
	let description = '';
	if (rawMeta) {
		const lines = rawMeta.split(/\r?\n/);
		let inTags = false;

		for (const line of lines) {
			// tags block
			if (/^\s*tags:\s*$/.test(line)) {
				inTags = true;
				continue;
			}
			if (inTags) {
				const m = line.match(/^\s*-\s*(.+)$/);
				if (m) {
					tags.push(m[1].trim());
					continue;
				}
				// end of tags block on first non-indented line
				if (/^\S/.test(line)) {
					inTags = false;
				}
			}

			const dateMatch = line.match(/^\s*published-date:\s*(.+)$/);
			if (dateMatch) {
				publishedDate = dateMatch[1].trim().replaceAll('"', '');
			}
			const titleMatch = line.match(/^\s*title:\s*(.+)$/);
			if (titleMatch) {
				title = titleMatch[1].trim().replaceAll('"', '');
			}
			const descriptionMatch = line.match(/^\s*description:\s*(.+)$/);
			if (descriptionMatch) {
				description = descriptionMatch[1].trim().replaceAll('"', '');
			}
		}
	}

	const html = $derived.by(() => {
		const baseHtml = marked(body, { async: false });
		return preprocessHtml ? preprocessHtml(baseHtml) : baseHtml;
	});
</script>

<PageTop containerClass="border-t-0" contentClass={"pt-4"}>
	<Breadcrumbs activeLinkClass="text-primary" contentClass={'w-full mb-12'} />
	<div class="mx-auto max-w-screen-lg">
		<div class="space-y-2">
			<T tag="h1" class="text-balance !font-bold">
				<!-- {m.EUDIW_Conformance_Interoperability_and_Marketplace()} -->
				{title}
			</T>
			<div class="flex flex-col gap-2 py-2">
				<T tag="p" class="text-primary text-balance !font-bold">
					{description}
				</T>
			</div>
		</div>
		{#if publishedDate}
			<div class="my-7 flex gap-2">
				<T tag="small" class="text-balance !font-normal">
					{'Published on:'}
				</T>
				<T tag="small" class="text-balance !font-bold">
					{publishedDate}
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
					<Button variant="outline" class="border-primary text-primary" href={''}>
						{tag}
					</Button>
				{/each}
			</div>
		{/if}
	</div>
</PageTop>
<!-- <div class="mx-auto p-8 py-12 border"> -->
<!-- <div class="prose prose-headings:font-serif prose-h1:text-3xl {className}">{@html body}</div> -->
<div class="bg-secondary">
	<PageContent>
		<div class="mx-auto max-w-screen-lg">
			<div class="prose prose-headings:font-serif prose-h1:text-3xl {className}">
				{@html html}
			</div>
		</div>
	</PageContent>
</div>

<!-- </div> -->

<style lang="postcss">
	/* Container: flex with small spacing */
	.breadcrumb {
		@apply flex space-x-2;
	}

	/* All non‐last <li> <a> links get primary color + hover underline */
	.breadcrumb > li:not(:last-child) > a {
		@apply text-primary hover:underline;
	}

	/* The last <li> has no styling on its <span> (and no hover, etc.) */
	.breadcrumb > li:last-child > span {
		/* No Tailwind @apply here—unstyled */
	}
</style>
