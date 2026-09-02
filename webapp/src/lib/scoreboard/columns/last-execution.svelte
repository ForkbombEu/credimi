<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { renderComponent } from '@/components/ui/data-table';
	import { m } from '@/i18n';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.latest_execution?.created,
		id: 'last_execution',
		header: renderComponent(EntityHeader, {
			label: m.scoreboard_last_run()
		}),
		sortField: 'latest_execution.created',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	import { fromStore } from 'svelte/store';

	import { currentUser } from '@/pocketbase';

	import * as EntityDisplay from '../entity-display';
	import { formatExecutionTimestamp } from '../extras/format-date';

	let { value }: Column.Props<typeof column> = $props();

	const user = fromStore(currentUser);
	const formatted = $derived(formatExecutionTimestamp(value, user.current?.Timezone));
</script>

{#if formatted}
	<time class="font-mono text-[11px] whitespace-nowrap" datetime={value}>{formatted}</time>
{:else}
	<EntityDisplay.Na />
{/if}
