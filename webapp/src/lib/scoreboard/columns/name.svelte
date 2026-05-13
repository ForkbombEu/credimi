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
		fn: (row) => row.expand.pipeline,
		id: 'name',
		header: renderComponent(EntityHeader, {
			data: entities.pipelines
		}),
		sortField: 'pipeline.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	import { getPath } from '$lib/utils';

	import A from '@/components/ui-custom/a.svelte';

	import * as EntityDisplay from '../entity-display';

	//

	let { value }: Column.Props<typeof column> = $props();

	const href = $derived(value ? `/hub/pipelines/${getPath(value)}` : null);
</script>

<div class="leading-none wrap-break-word whitespace-normal">
	{#if href && value}
		<A {href} class="text-xs font-bold">{value.name}</A>
	{:else}
		<EntityDisplay.Na />
	{/if}
</div>
