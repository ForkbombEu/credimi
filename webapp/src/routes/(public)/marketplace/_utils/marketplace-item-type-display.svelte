<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';

	import type { MarketplaceItemDisplayData } from '.';

	type Props = {
		data: MarketplaceItemDisplayData;
	};

	let { data }: Props = $props();
</script>

<div class="flex items-center gap-1">
	<div class="{data?.bgClass} size-1.5 rounded-full"></div>
	<T class="{data?.textClass} text-sm">{data?.label}</T>
</div>
