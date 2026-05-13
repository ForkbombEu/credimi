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
		id: 'conformance_checks',
		header: renderComponent(EntityHeader, {
			data: entities.conformance_checks,
			plurality: 'plural'
		}),
		fn: (row) => EntityDisplay.fromConformancePaths(row.conformance_checks ?? []),
		sortField: 'use_case_verifications.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="full" />
