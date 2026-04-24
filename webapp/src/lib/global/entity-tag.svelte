<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EntityData } from './entities';

	type Props = {
		data: EntityData;
		trimLabel?: boolean;
		hideIcon?: boolean;
	};

	let { data, trimLabel = false, hideIcon = false }: Props = $props();

	const IconComponent = $derived(data.icon);
</script>

<div
	class="inline-flex items-center gap-1.5 rounded-sm border px-2.5 py-1 text-xs font-bold {data
		.classes.bg} {data.classes.border} {data.classes.text}"
>
	{#if IconComponent && !hideIcon}
		<IconComponent class="h-4 w-4" />
	{/if}
	{#if trimLabel}
		{data.labels.singular.slice(0, 3)}.
	{:else}
		{data.labels.singular}
	{/if}
</div>
