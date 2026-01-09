<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpIcon, TrashIcon } from 'lucide-svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';

	import { type EnrichedStep, type StepsBuilder } from '../steps-builder.svelte.js';
	import StepCardDisplay from './step-card-display.svelte';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
	};

	let { builder, step, index }: Props = $props();
</script>

<StepCardDisplay
	{step}
	onContinueOnErrorChange={(checked) => builder.setContinueOnError(index, checked)}
>
	{#snippet topRight()}
		<div
			class="flex items-center gap-1 pr-1 opacity-30 transition-opacity group-hover:opacity-100"
		>
			<IconButton
				icon={ArrowUpIcon}
				variant="ghost"
				size="xs"
				onclick={() => builder.shiftStep(index, -1)}
				disabled={!builder.canShiftStep(index, -1)}
			/>
			<IconButton
				icon={ArrowDownIcon}
				variant="ghost"
				size="xs"
				onclick={() => builder.shiftStep(index, 1)}
				disabled={!builder.canShiftStep(index, 1)}
			/>
			<IconButton
				icon={TrashIcon}
				variant="ghost"
				size="xs"
				onclick={() => builder.deleteStep(index)}
			/>
		</div>
	{/snippet}
</StepCardDisplay>
