<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MarketplaceItemDisplayData } from '.';

	type Props = {
		data: MarketplaceItemDisplayData;
	};

	let { data }: Props = $props();

	const IconComponent = $derived(data?.icon);
</script>

<div
	class="inline-flex items-center gap-1.5 rounded-sm border px-2.5 py-1 text-xs font-bold {data?.backgroundClass} {data?.outlineClass} {data?.textClass}"
>
	{#if IconComponent}
		<IconComponent class="h-4 w-4" />
	{/if}
	{data?.label}
</div>
