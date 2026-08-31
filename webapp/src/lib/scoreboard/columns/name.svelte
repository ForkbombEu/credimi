<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';

	//

	export const column = Column.define({
		fn: (row) => ({
			pipeline: row.expand.pipeline,
			stats: {
				total: row.total_runs ?? 0,
				successes: row.total_successes ?? 0,
				percent: row.success_rate
			}
		}),
		id: 'name',
		header: renderComponent(EntityHeader, {
			data: entities.pipelines,
			hideIcon: true
		}),
		sortField: 'pipeline.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	import { getPath } from '$lib/utils';

	import A from '@/components/ui-custom/a.svelte';
	import { m } from '@/i18n';

	import * as EntityDisplay from '../entity-display';
	import ScorePill from '../score-pill.svelte';

	//

	let { value }: Column.Props<typeof column> = $props();

	const pipeline = $derived(value.pipeline);
	const href = $derived(pipeline ? `/hub/pipelines/${getPath(pipeline)}` : null);
</script>

<div class="flex flex-col gap-1.5">
	{#if href && pipeline}
		{#if pipeline.published}
			<A {href} class="text-sm font-semibold whitespace-nowrap">{pipeline.name}</A>
		{:else}
			<span class="text-sm whitespace-nowrap">{pipeline.name}</span>
		{/if}
	{:else}
		<EntityDisplay.Na />
	{/if}
	<div class="flex items-center gap-2">
		<ScorePill percent={value.stats.percent} total={value.stats.total} />
		<span class="text-xs whitespace-nowrap text-muted-foreground">
			{m.scoreboard_runs_passed({
				successes: value.stats.successes,
				total: value.stats.total
			})}
		</span>
	</div>
</div>
