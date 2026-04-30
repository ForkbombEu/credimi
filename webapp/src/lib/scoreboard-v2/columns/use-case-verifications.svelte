<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities, EntityTag } from '$lib/global';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.expand.use_case_verifications ?? [],
		id: 'use_case_verifications',
		header: Column.header(EntityTag, {
			data: entities.use_cases_verifications
		})
	});
</script>

<script lang="ts">
	import SmallList from './partials/small-list.svelte';

	let { value }: Column.Props<typeof column> = $props();
</script>

<SmallList records={value} />
