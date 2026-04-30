<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities, EntityTag } from '$lib/global';

	import * as Column from '../column';

	export const column = Column.define({
		fn: (row) => row.expand.credentials ?? [],
		id: 'credentials',
		header: Column.header(EntityTag, {
			data: entities.credentials
		})
	});
</script>

<script lang="ts">
	import SmallList from './partials/small-list.svelte';

	//

	let { value }: Column.Props<typeof column> = $props();
</script>

<SmallList records={value} />
