<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { scoreBand } from './score-band';

	//

	type Props = {
		percent: number | null | undefined;
		total: number;
	};

	let { percent, total }: Props = $props();

	const band = $derived(scoreBand(percent, total));
</script>

{#if band.key === 'none'}
	<span class="text-sm text-muted-foreground">—</span>
{:else}
	<span
		class={[
			'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] leading-none font-semibold whitespace-nowrap',
			band.pill
		]}
	>
		<span class={['size-1.5 shrink-0 rounded-full', band.dot]} aria-hidden="true"></span>
		{percent}% {band.label}
	</span>
{/if}
