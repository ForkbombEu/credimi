<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.expand.mobile_runners ?? [],
		id: 'runners',
		header: m.Runners()
	});
</script>

<script lang="ts">
	import Tooltip from '@/components/ui-custom/tooltip.svelte';

	import Na from './partials/na.svelte';

	let { value }: Column.Props<typeof column> = $props();
</script>

<div class="flex flex-col items-start">
	{#each value as item (item)}
		<Tooltip>
			<p class="max-w-[15ch] truncate text-xs">{item.name.trim()}</p>

			{#snippet content()}
				<p>{item.description}</p>
			{/snippet}
		</Tooltip>
	{:else}
		<Na />
	{/each}
</div>
