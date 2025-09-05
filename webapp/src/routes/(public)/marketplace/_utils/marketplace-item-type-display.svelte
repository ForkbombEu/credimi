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
	class="inline-flex items-center gap-1.5 rounded-sm border px-2.5 py-1 text-xs font-medium {data?.backgroundClass} {data?.outlineClass} {data?.textClass}"
>
	{#if IconComponent}
		<IconComponent class="h-3 w-3" />
	{/if}
	{data?.label}
</div>
