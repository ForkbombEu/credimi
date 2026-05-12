<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { Snippet } from 'svelte';

	import BackButton from '$lib/layout/back-button.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';

	import Avatar from '@/components/ui-custom/avatar.svelte';

	import LayoutWithToc from '../../../../[...path]/_partials/_utils/layout-with-toc.svelte';

	//

	type Props = {
		top: Snippet;
		content: Snippet;
		tocSections: IndexItem[];
		logo?: string;
	};

	let { top, content, tocSections, logo }: Props = $props();
</script>

<PageTop contentClass="!space-y-4">
	<BackButton href="/marketplace" />

	<div class="flex items-center gap-6">
		{#if logo}
			<Avatar src={logo} class="size-32 rounded-md border" hideIfLoadingError />
		{/if}

		<div class="space-y-1">
			{@render top?.()}
		</div>
	</div>
</PageTop>

<LayoutWithToc sections={tocSections}>
	{@render content()}
</LayoutWithToc>
