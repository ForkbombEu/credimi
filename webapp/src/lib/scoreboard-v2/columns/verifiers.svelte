<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import * as Column from '../column';
	import Avatar from './partials/avatar.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.verifiers ?? [],
		id: 'verifiers',
		header: 'Verifiers'
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div class="flex flex-col items-end gap-1">
	{#each value as item (item.id)}
		<Avatar record={item} link />
	{/each}
</div>
