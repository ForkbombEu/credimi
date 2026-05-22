<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpIcon, CopyPlus, PencilIcon, TrashIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import type { StepsBuilder } from '../steps-builder.svelte.js';
	import type { EnrichedStep } from '../types';

	import StepCardDisplay from './step-card-display.svelte';
	import { isStepEditable } from './utils.js';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
		editing?: boolean;
	};

	let { builder, step, index, editing = false }: Props = $props();

	const editable = $derived(isStepEditable(step));
</script>

<StepCardDisplay
	{step}
	{editing}
	onContinueOnErrorChange={(checked) => builder.setContinueOnError(index, checked)}
>
	{#snippet topRight()}
		<div
			class="flex items-center gap-1 pr-1 opacity-30 transition-opacity group-hover:opacity-100"
		>
			{#if editable}
				<IconButton
					icon={PencilIcon}
					variant="ghost"
					size="xs"
					onclick={() => builder.initEditStep(index)}
				/>
			{/if}
			<IconButton
				icon={CopyPlus}
				variant="ghost"
				size="xs"
				tooltip={m.Clone()}
				onclick={() => builder.cloneStep(index)}
			/>
			<IconButton
				icon={TrashIcon}
				variant="ghost"
				size="xs"
				onclick={() => builder.deleteStep(index)}
			/>
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
		</div>
	{/snippet}
</StepCardDisplay>
