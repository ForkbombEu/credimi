<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';
	import SmallList from './partials/small-list.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.custom_integrations ?? [],
		id: 'custom_integrations',
		header: renderComponent(EntityHeader, {
			data: entities.custom_checks,
			plurality: 'plural'
		})
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<SmallList records={value} />
