<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';
	import BaseHeader from './headers/base-header.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.mobile_runners ?? [],
		id: 'runners',
		header: Column.header(BaseHeader, {
			header: m.Runners()
		})
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

{#each value as item (item)}
	<p class="max-w-[15ch] truncate text-xs">{item.name}</p>
{/each}
