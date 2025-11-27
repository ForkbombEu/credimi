<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { Snippet } from 'svelte';

	import BackButton from '$lib/layout/back-button.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';

	import { m } from '@/i18n/index.js';

	import LayoutWithToc from '../../../../[...path]/_partials/_utils/layout-with-toc.svelte';

	//

	type Props = {
		top: Snippet;
		content: Snippet;
		tocSections: IndexItem[];
	};

	let { top, content, tocSections }: Props = $props();
</script>

<PageTop contentClass="!space-y-4">
	<BackButton href="/marketplace">
		{m.Back_to_marketplace()}
	</BackButton>

	<div class="space-y-1">
		{@render top?.()}
	</div>
</PageTop>

<LayoutWithToc sections={tocSections}>
	{@render content()}
</LayoutWithToc>
