<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';

	import HTML from '@/components/ui-custom/renderHTML.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Badge from '@/components/ui/badge/badge.svelte';
	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';

	let { data } = $props();
	const { news } = $derived(data);

	const tags = $derived(
		news.tags
			.split('#')
			.filter(Boolean)
			.map((tag) => tag.trim())
	);
</script>

<PageTop>
	<BackButton href="/news" />

	<div class="flex flex-col gap-4">
		<T tag="h1">{news.title}</T>
		<HTML class="text-primary" content={news.summary} />
	</div>
	<div>
		<T tag="small" class="text-muted-foreground">
			{m.published_on()}
			<span class="text-black">
				{new Date(news.updated).toLocaleString()}
			</span>
		</T>
	</div>

	<!-- TAGS -->
	{#if tags.length > 0}
		<div class="flex flex-row items-center justify-start gap-2">
			{#each tags as tag (tag)}
				<Badge variant="outline" class="border-primary text-primary">{tag}</Badge>
			{/each}
		</div>
	{/if}

	<!-- LINKS -->
	<div class="flex flex-col items-start justify-start gap-2">
		<T tag="small" class="text-muted-foreground">Links:</T>
		<div class="flex flex-row items-center justify-start gap-2">
			{#if news.diff}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={news.diff} target="_blank">
					<Button size="sm">{m.differences()}</Button>
				</a>
			{/if}
			{#if news.refer}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={news.refer} target="_blank">
					<Button size="sm">{m.referrer()}</Button>
				</a>
			{/if}
		</div>
	</div>
</PageTop>

<PageContent class="grow bg-secondary" contentClass="flex gap-12 items-start">
	<div class="space-y-12">
		<div>
			<PageHeader title="Key differences" id="key_differences" />
			<RenderMd class="prose" content={news.key_differences} />
		</div>
		<div>
			<PageHeader title="news" id="news" />
			<HTML class="prose" content={news.news} />
		</div>
	</div>
</PageContent>
