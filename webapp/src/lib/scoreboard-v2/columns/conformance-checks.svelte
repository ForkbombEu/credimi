<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';
	import Na from './partials/na.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.conformance_checks,
		id: 'conformance_checks',
		header: m.Conformance_Checks()
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div>
	{#each value as item (item)}
		<p class="max-w-[15ch] truncate text-xs">{item}</p>
	{:else}
		<Na />
	{/each}
</div>
