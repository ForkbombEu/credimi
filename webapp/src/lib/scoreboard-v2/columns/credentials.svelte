<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';

	export const column = Column.define({
		fn: (row) => row.expand.credentials ?? [],
		id: 'credentials',
		header: renderComponent(EntityHeader, {
			data: entities.credentials,
			plurality: 'plural'
		})
	});
</script>

<script lang="ts">
	import SmallList from './partials/small-list.svelte';

	//

	let { value }: Column.Props<typeof column> = $props();
</script>

<SmallList records={value} />
