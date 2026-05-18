<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ScoreboardRow } from '../types';

	import * as EntityDisplay from '../entity-display';

	type Props = {
		results: ScoreboardRow;
	};

	let { results }: Props = $props();

	const items = $derived(EntityDisplay.buildPipelineSummaryItems(results));
</script>

{#if items.length > 0}
	<EntityDisplay.List {items} layout="compact" />
{/if}
