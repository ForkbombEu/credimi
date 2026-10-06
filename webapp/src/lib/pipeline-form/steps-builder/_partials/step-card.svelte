<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { ArrowDownIcon, ArrowUpIcon, CopyPlus, PencilIcon, TrashIcon } from '@lucide/svelte';
	import { comp } from '$lib/renderable';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import type { InCardHostChrome } from './in-card-host-chrome.js';

	import ContinueOnErrorFooter from './continue-on-error-footer.svelte';
	import InCardHost from './in-card-host.svelte';
	import { isStepEditable } from './utils.js';

	type Props = InCardHostChrome & {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
		onShift: (change: number) => void;
	};

	let { builder, step, index, onShift, ...chrome }: Props = $props();

	const editable = $derived(isStepEditable(step));
	const actionsDisabled = $derived(builder.isFormMode);
</script>

<InCardHost
	{builder}
	{step}
	{...chrome}
	footer={comp(ContinueOnErrorFooter, {
		step,
		onCheckedChange: (checked) => builder.setContinueOnError(index, checked)
	})}
>
	{#snippet topRight()}
		{#if editable}
			<IconButton
				icon={PencilIcon}
				variant="ghost"
				size="xs"
				class="pointer-events-auto"
				onclick={() => builder.initEditStep(index)}
			/>
		{/if}
		<IconButton
			icon={CopyPlus}
			variant="ghost"
			size="xs"
			tooltip={m.Clone()}
			disabled={actionsDisabled}
			onclick={() => builder.cloneStep(index)}
		/>
		<IconButton
			icon={TrashIcon}
			variant="ghost"
			size="xs"
			disabled={actionsDisabled}
			onclick={() => builder.deleteStep(index)}
		/>
		<IconButton
			icon={ArrowUpIcon}
			variant="ghost"
			size="xs"
			onclick={() => onShift(-1)}
			disabled={actionsDisabled || !builder.canShiftStep(index, -1)}
		/>
		<IconButton
			icon={ArrowDownIcon}
			variant="ghost"
			size="xs"
			onclick={() => onShift(1)}
			disabled={actionsDisabled || !builder.canShiftStep(index, 1)}
		/>
	{/snippet}
</InCardHost>
