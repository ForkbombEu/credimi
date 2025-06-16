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

	const { data } = $props();
	let { attributes, body } = data;
</script>

<PageTop containerClass="border-t-0" contentClass={'pt-4'}>
	<Breadcrumbs />
	{#if attributes}
		{@const { title, description, tags, date } = attributes}
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
			<div class="prose prose-headings:font-serif prose-h1:text-3xl">
				{@html body}
			</div>
		</div>
	</PageContent>
</div>
