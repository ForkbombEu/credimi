<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { ConformanceChecks } from '$lib';
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';
	import Na from './partials/na.svelte';

	//

	export const column = Column.define({
		id: 'conformance_checks',
		header: renderComponent(EntityHeader, {
			data: entities.conformance_checks,
			plurality: 'plural'
		}),
		fn: (row) => ConformanceChecks.groupPathsByStandard(row.conformance_checks ?? [])
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div>
	{#each value as item (item)}
		<p class="text-xs font-bold">{item.title}</p>
		<ul class="list-inside list-disc">
			{#each item.items as x (x)}
				<li class="max-w-[35ch] truncate text-xs">
					{x}
				</li>
			{/each}
		</ul>
	{:else}
		<Na />
	{/each}
</div>
