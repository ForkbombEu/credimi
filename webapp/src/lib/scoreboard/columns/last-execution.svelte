<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.expand.latest_successful_execution?.created,
		id: 'last_execution',
		header: m.scoreboard_last_run(),
		sortField: 'latest_successful_execution.created'
	});
</script>

<script lang="ts">
	import { fromStore } from 'svelte/store';

	import { currentUser } from '@/pocketbase';

	import * as EntityDisplay from '../entity-display';

	//

	let { value }: Column.Props<typeof column> = $props();

	const user = fromStore(currentUser);

	const formatted = $derived.by(() => {
		if (!value) return undefined;

		const parsed = new Date(value);
		if (Number.isNaN(parsed.getTime())) return undefined;

		const timeZone = user.current?.Timezone;

		return parsed.toLocaleString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit',
			...(timeZone ? { timeZone } : {})
		});
	});
</script>

{#if formatted}
	<time class="text-xs whitespace-nowrap" datetime={value}>{formatted}</time>
{:else}
	<EntityDisplay.Na />
{/if}
