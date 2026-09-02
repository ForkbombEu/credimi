<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EntityData } from '$lib/global';

	import SortHeaderPill from '$lib/scoreboard/sort-header-pill.svelte';

	import type { HeaderAlign } from './alignment';

	import { getHeaderContext } from './header-context-provider.svelte';

	//

	type Props = {
		data?: EntityData;
		trimLabel?: boolean;
		hideIcon?: boolean;
		plurality?: 'singular' | 'plural';
		align?: HeaderAlign;
		/** For non-entity columns (e.g. Runners, Last run): plain label, same typography */
		label?: string;
	};

	let { align = 'left', label: labelOverride, ...props }: Props = $props();

	const ctx = getHeaderContext();

	const label = $derived(
		labelOverride ??
			(props.plurality === 'singular'
				? props.data!.labels.singular
				: props.data!.labels.plural)
	);
</script>

<div
	class={[
		'flex items-center gap-1',
		{
			'justify-start': align === 'left',
			'justify-center': align === 'center',
			'justify-end': align === 'right'
		}
	]}
>
	<span
		class="inline-flex items-center gap-1.5 text-[11px] font-medium tracking-[0.08em] whitespace-nowrap text-primary uppercase"
	>
		{#if props.data?.icon && !props.hideIcon}
			<props.data.icon class="size-3.5" />
		{/if}
		{label}
	</span>
	{#if ctx.header.column.getCanSort() && ctx.header.column.columnDef.meta?.manualPillPositioning}
		<SortHeaderPill header={ctx.header} table={ctx.table} />
	{/if}
</div>
