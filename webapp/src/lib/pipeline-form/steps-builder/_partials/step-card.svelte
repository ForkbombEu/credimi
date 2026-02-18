<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpIcon, PencilIcon, TrashIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';

	import type { StepsBuilder } from '../steps-builder.svelte.js';
	import type { EnrichedStep } from '../types';

	import StepCardDisplay from './step-card-display.svelte';
	import { getStepConfig } from './utils.js';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
	};

	let { builder, step, index }: Props = $props();

	const config = $derived(getStepConfig(step));
	const EditComponent = $derived(config?.EditComponent);

	let isEditSheetOpen = $state(false);
</script>

<StepCardDisplay
	{step}
	onContinueOnErrorChange={(checked) => builder.setContinueOnError(index, checked)}
>
	{#snippet topRight()}
		<div
			class="flex items-center gap-1 pr-1 opacity-30 transition-opacity group-hover:opacity-100"
		>
			{#if EditComponent}
				<IconButton
					icon={PencilIcon}
					variant="ghost"
					size="xs"
					onclick={() => (isEditSheetOpen = true)}
				/>
			{/if}
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

<Sheet bind:open={isEditSheetOpen} hideTrigger>
	{#snippet content({ closeSheet })}
		{#if EditComponent}
			<EditComponent data={step[1]} closeDialog={closeSheet} />
		{/if}
	{/snippet}
</Sheet>
