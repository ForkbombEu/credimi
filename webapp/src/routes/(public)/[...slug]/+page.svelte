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

	const { data } = $props();
	let { attributes, body } = data;
</script>

<PageTop containerClass="border-t-0" contentClass="pt-8 !space-y-12">
	<Breadcrumbs />

	{#if attributes}
		{@const { title, description, tags, date } = attributes}
		<div class="max-w-prose space-y-6">
			<div class="space-y-2">
				{#if date}
					<div class="text-muted-foreground flex gap-1">
						<T tag="small" class="text-balance !font-normal">
							{'Published on:'}
						</T>
						<T tag="small" class="text-balance !font-semibold">
							{date.toLocaleDateString()}
						</T>
					</div>
				{/if}

				<T tag="h1" class="text-balance !font-bold">
					{title}
				</T>
			</div>

			<T tag="p" class="text-primary text-balance !font-bold">
				{description}
			</T>

			{#if tags.length}
				<div class="flex items-center gap-2">
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
	<div class="mx-auto max-w-screen-lg">
		<div class="prose prose-h1:text-3xl">
			{@html body}
		</div>
	</div>
</PageContent>
