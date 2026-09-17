<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';
	import { m } from '@/i18n';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';
	import EntityHeader from './headers/entity-header.svelte';

	// Presentations column: verifier logos only. Use-case nesting belongs in
	// card/summary via fromPresentationSummaryItems, not this table column.
	export const column = Column.define({
		fn: (row) =>
			EntityDisplay.fromPocketbaseEntities(
				row.expanded_data?.verifiers ?? [],
				entities.verifiers
			),
		id: 'verifiers',
		header: renderComponent(EntityHeader, {
			label: m.Presentations()
		}),
		sortField: 'verifiers.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="logos" />
