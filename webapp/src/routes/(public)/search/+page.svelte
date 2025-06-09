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
	import type { PageData } from './$types';
	import Fuse from 'fuse.js';
	import PageCard from './page-list-item.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import * as Popover from '@/components/ui/popover/index.js';
	import { Checkbox } from '@/components/ui/checkbox';
	import { Label } from '@/components/ui/label';
	import { onMount } from 'svelte';

	export const AVAILABLE_FILTERS = ['news/', 'documentation/', 'tags'] as const;
	type FilterName = (typeof AVAILABLE_FILTERS)[number];

	interface Page {
		slug: string;
		title: string;
		description: string;
		date: string;
		tags: string[];
		updatedOn: string;
		rawBody: string;
		html: string;
	}

	const fuseOptions = {
		keys: [
			{ name: 'title', weight: 0.4 },
			{ name: 'description', weight: 0.3 },
			{ name: 'tags', weight: 0.2 },
			{ name: 'rawBody', weight: 0.1 }
		],
		includeScore: true,
		threshold: 0.3
	};

	const { data } = $props<{ data: PageData }>();

	let paramTag = page.url.searchParams.get('tag') ?? null;

	onMount(() => {
		if (paramTag) {
			selectedFilters = ['tags'];
			searchText = paramTag;
		}
	});

	let searchText = $state('');
	let fuseStore = $derived.by(() => {
		return new Fuse(data.pages, fuseOptions);
	});
	let selectedFilters = $state<FilterName[]>([]);
	let filteredPages = $derived.by(() => {
		const search = searchText.trim().toLowerCase();
		if (search === '') {
			return [];
		}
		const fuseResults = fuseStore.search(search).map((r) => r.item as Page);
		const pathFilters = selectedFilters.filter((f) => f.endsWith('/'));
		const hasTagFilter = selectedFilters.includes('tags');
		let results = fuseResults;
		if (pathFilters.length > 0) {
			results = results.filter((p) =>
				pathFilters.some((prefix) => p.slug.startsWith(prefix))
			);
		}
		if (hasTagFilter) {
			results = results.filter((p) => p.tags.some((t) => t.toLowerCase().includes(search)));
		}
		return results;
	}) as Page[];

	const filterName = $derived.by(() => {
		if (selectedFilters.length === 0) {
			return '';
		}
		const hasTags = selectedFilters.includes('tags');
		const pathFilters = selectedFilters.filter((f) => f.endsWith('/'));
		const pathNames = pathFilters.map(fixFilterName);
		let joinedPaths: string;
		if (pathNames.length === 0) {
			joinedPaths = '';
		} else if (pathNames.length === 1) {
			joinedPaths = pathNames[0];
		} else {
			const allButLast = pathNames.slice(0, -1).join(', ');
			const last = pathNames[pathNames.length - 1];
			joinedPaths = `${allButLast} and ${last}`;
		}

		if (hasTags) {
			if (joinedPaths) {
				return `Tags for ${joinedPaths}`;
			} else {
				return `Tags`;
			}
		} else {
			return joinedPaths;
		}
	});

	function fixFilterName(filter: string) {
		const raw = filter.endsWith('/') ? filter.slice(0, -1) : filter;
		return raw.charAt(0).toUpperCase() + raw.slice(1);
	}
</script>

<PageTop>
	<T tag="h1">{m.Search()}</T>
	{#if filterName}
		<T tag="h4" class="text-primary">{`${m.Filters()}: ${filterName}`}</T>
	{/if}
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

		<Popover.Root>
			<Popover.Trigger
				class={`${buttonVariants({ variant: 'outline' })} border-primary bg-secondary`}
			>
				{m.Filters()}
			</Popover.Trigger>

			<Popover.Content class="w-80">
				<ul class="space-y-2">
					{#each AVAILABLE_FILTERS as f}
						<li>
							<div class="flex items-center gap-2">
								<Checkbox
									id={f}
									name={f}
									value={f}
									checked={selectedFilters.includes(f)}
									onCheckedChange={(v) => {
										if (v) {
											selectedFilters.push(f);
										} else {
											selectedFilters = selectedFilters.filter(
												(filt) => filt !== f
											);
										}
									}}
								/>
								<Label for={f} class="text-md hover:cursor-pointer">
									{(f.charAt(0).toUpperCase() + f.slice(1)).replace('/', '')}
								</Label>
							</div>
						</li>
					{/each}
				</ul>
			</Popover.Content>
		</Popover.Root>
	</div>
</PageTop>

<PageContent class="bg-secondary grow">
	{#if filteredPages.length > 0}
		<PageGrid>
			{#each filteredPages as page}
				<PageCard
					{...page}
					{searchText}
					onTagChange={(tag: string) => {
						searchText = tag;
						selectedFilters = ['tags'];
					}}
				/>
			{/each}
		</PageGrid>
	{:else}
		<T tag="p">{m.No_results_found()}</T>
	{/if}
</PageContent>
