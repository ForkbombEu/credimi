<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';
	import BaseHeader from './headers/base-header.svelte';
	import SmallList from './partials/small-list.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.custom_integrations ?? [],
		id: 'custom_integrations',
		header: Column.header(BaseHeader, {
			header: m.Custom_integrations()
		})
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<SmallList records={value} />
