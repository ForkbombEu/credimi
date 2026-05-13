<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';
	import EntityHeader from './headers/entity-header.svelte';

	export const column = Column.define({
		fn: (row) =>
			EntityDisplay.fromPocketbaseEntities(row.expand.custom_integrations ?? [], entities.custom_checks),
		id: 'custom_integrations',
		header: renderComponent(EntityHeader, {
			data: entities.custom_checks,
			plurality: 'plural'
		}),
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="links-only" />
