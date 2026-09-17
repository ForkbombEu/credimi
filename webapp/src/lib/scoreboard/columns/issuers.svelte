<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { renderComponent } from '@/components/ui/data-table';
	import { m } from '@/i18n';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';
	import EntityHeader from './headers/entity-header.svelte';

	export const column = Column.define({
		fn: (row) =>
			EntityDisplay.fromIssuanceItems(
				row.expanded_data?.issuers ?? [],
				row.expanded_data?.credentials ?? []
			),
		id: 'issuers',
		header: renderComponent(EntityHeader, {
			label: m.Issuance(),
			align: 'right'
		}),
		sortField: 'issuers.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="logos" />
