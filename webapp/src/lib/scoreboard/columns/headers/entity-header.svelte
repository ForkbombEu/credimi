<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { EntityTag } from '$lib/global';
	import SortHeaderPill from '$lib/scoreboard/sort-header-pill.svelte';

	import type { HeaderAlign } from './alignment';

	import { getHeaderContext } from './header-context-provider.svelte';

	//

	type Props = ComponentProps<typeof EntityTag> & {
		align?: HeaderAlign;
	};

	let { align = 'left', ...props }: Props = $props();

	const ctx = getHeaderContext();
</script>

<div
	class={[
		'relative flex items-center',
		{
			'justify-start': align === 'left',
			'justify-center': align === 'center',
			'justify-end': align === 'right'
		}
	]}
>
	<EntityTag {...props} />
	<div class="absolute top-0 right-0 translate-x-3 -translate-y-1">
		{#if ctx.header.column.getCanSort() && ctx.header.column.columnDef.meta?.manualPillPositioning}
			<SortHeaderPill header={ctx.header} table={ctx.table} />
		{/if}
	</div>
</div>
