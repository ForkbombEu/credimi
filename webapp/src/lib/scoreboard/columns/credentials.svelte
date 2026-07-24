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
			EntityDisplay.fromPocketbaseEntities(
				row.expand.credentials ?? [],
				entities.credentials
			),
		id: 'credentials',
		header: renderComponent(EntityHeader, {
			data: entities.credentials,
			plurality: 'plural'
		}),
		sortField: 'credentials.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.LeadingAvatarList items={value} />
