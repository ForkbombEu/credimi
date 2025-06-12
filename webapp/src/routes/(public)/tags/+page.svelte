<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import { Button, buttonVariants } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import PageTop from '$lib/layout/pageTop.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { X } from 'lucide-svelte';
	import { String } from 'effect';
	import PageCard from './page-list-item.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import * as Popover from '@/components/ui/popover/index.js';
	import { Checkbox } from '@/components/ui/checkbox';
	import { Label } from '@/components/ui/label';
	import { onMount } from 'svelte';

	const { data } = $props();
	const {pages } = data; 
	
	let paramTag = page.url.searchParams.get('search') ?? null;

	onMount(() => {
		if (paramTag) {
			searchText = paramTag;
		}
	});

	let searchText = $state('');

	let results = $derived.by(() => {
		const search = searchText.trim().toLowerCase();
		if (search === '') {
			return pages;
		}
		return pages.filter((page) => page.tags.includes(search));
	});
</script>

<PageTop>
	<T tag="h1">{m.Search_tags()}</T>
	<div class="flex items-center gap-2">
		<div class="relative flex grow">
			<Input
				bind:value={searchText}
				placeholder={m.Search()}
				class="border-primary bg-secondary"
			/>
			{#if String.isString(searchText)}
				<Button
					onclick={() => {
						searchText = '';
					}}
					class="absolute right-1 top-1 size-8"
					variant="ghost"
				>
					<Icon src={X} size="" />
				</Button>
			{/if}
		</div>
	</div>
</PageTop>

<PageContent class="bg-secondary grow">
	{#if results.length > 0}
		<PageGrid>
			{#each results as page}
				<PageCard
					{...page}
					onTagChange={(tag: string) => {
						searchText = tag;
					}}
				/>
			{/each}
		</PageGrid>
	{:else}
		<T tag="p">{m.No_results_found()}</T>
	{/if}
</PageContent>
