<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';
	import Na from './partials/na.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.conformance_checks,
		id: 'conformance_checks',
		header: renderComponent(EntityHeader, {
			data: entities.conformance_checks,
			plurality: 'plural'
		})
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div>
	{#each value as item (item)}
		<p class="max-w-[40ch] truncate text-xs">{item}</p>
	{:else}
		<Na />
	{/each}
</div>
