<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';
	import Avatar from './partials/avatar.svelte';
	import Na from './partials/na.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.issuers ?? [],
		id: 'issuers',
		header: renderComponent(EntityHeader, {
			data: entities.credential_issuers,
			trimLabel: true,
			hideIcon: true,
			align: 'right'
		})
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div class="flex flex-col items-end gap-1">
	{#each value as item (item.id)}
		<Avatar record={item} link />
	{:else}
		<Na />
	{/each}
</div>
