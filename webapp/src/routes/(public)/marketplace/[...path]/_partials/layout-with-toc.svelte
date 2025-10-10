<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import PageContent from '$lib/layout/pageContent.svelte';
	import PageIndex, { type IndexItem } from '$lib/layout/pageIndex.svelte';

	type Props = {
		sections: IndexItem[];
		children: Snippet;
	};

	let { sections, children }: Props = $props();
</script>

<PageContent class="bg-secondary grow" contentClass="flex flex-col md:flex-row gap-12 items-start">
	<PageIndex {sections} class="top-5 md:sticky" />

	<div class="grow space-y-16">
		{@render children()}
	</div>
</PageContent>
