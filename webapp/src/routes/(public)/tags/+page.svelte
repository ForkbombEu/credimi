<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageTop from '$lib/layout/pageTop.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import PageCard from './page-list-item.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import { getTagTranslation } from '$lib/content/tags-i18n';

	const { data } = $props();
	const pages = $derived(data.pages);
	const tag = $derived(data.tag);
</script>

<PageTop containerClass="border-t-0">
	<T tag="h1">{getTagTranslation(tag)}</T>
</PageTop>

<PageContent class="bg-secondary grow">
	{#if pages.length > 0}
		<PageGrid>
			{#each pages as page}
				<PageCard {...page} />
			{/each}
		</PageGrid>
	{:else}
		<T tag="p">{m.No_results_found()}</T>
	{/if}
</PageContent>
