<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PipelineFinallyCondition } from '$lib/pipeline/types';
	import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { ArrowDownIcon, ArrowUpIcon, CopyPlus, PencilIcon, TrashIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import { isStepEditable, StepCardDisplay } from './index.js';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
		editing?: boolean;
		followUpCondition?: PipelineFinallyCondition;
	};

	let { builder, step, index, editing = false, followUpCondition }: Props = $props();

	const editable = $derived(isStepEditable(step));
	const actionsDisabled = $derived(builder.isFormMode);
	const isFollowUp = $derived(followUpCondition !== undefined);

	const conditionOptions: { value: PipelineFinallyCondition; label: () => string }[] = [
		{ value: 'always', label: () => m.Always() },
		{ value: 'on_success', label: () => m.On_success() },
		{ value: 'on_failure', label: () => m.On_failure() }
	];
</script>

<StepCardDisplay
	{step}
	{editing}
	onContinueOnErrorChange={(checked) => builder.setContinueOnError(index, checked)}
	showContinueOnError={!isFollowUp}
>
	{#snippet topRight()}
		{#if isFollowUp}
			<div
				class={[
					'flex items-center gap-1 pr-1 transition-opacity',
					actionsDisabled ? 'opacity-30' : 'opacity-30 group-hover:opacity-100'
				]}
			>
				{#if editable}
					<IconButton
						icon={PencilIcon}
						variant="ghost"
						size="xs"
						onclick={() => builder.initEditFollowUp(index)}
					/>
				{/if}
				<IconButton
					icon={TrashIcon}
					variant="ghost"
					size="xs"
					disabled={actionsDisabled}
					onclick={() => builder.deleteFollowUp(index)}
				/>
			</div>
		{:else}
			<div
				class={[
					'flex items-center gap-1 pr-1 transition-opacity',
					actionsDisabled ? 'opacity-30' : 'opacity-30 group-hover:opacity-100'
				]}
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
					onclick={() => builder.shiftStep(index, -1)}
					disabled={actionsDisabled || !builder.canShiftStep(index, -1)}
				/>
				<IconButton
					icon={ArrowDownIcon}
					variant="ghost"
					size="xs"
					onclick={() => builder.shiftStep(index, 1)}
					disabled={actionsDisabled || !builder.canShiftStep(index, 1)}
				/>
			</div>
		{/if}
	{/snippet}

	{#snippet bottom()}
		{#if isFollowUp && followUpCondition}
			<div class="flex flex-wrap items-center gap-x-2 gap-y-1 bg-slate-50 px-3 py-1.5">
				<div class="inline-flex rounded-md border bg-background p-0.5">
					{#each conditionOptions as option (option.value)}
						<button
							type="button"
							class={[
								'rounded-sm px-2 py-1 text-[11px] font-medium transition-colors',
								option.value === followUpCondition
									? 'bg-muted text-foreground'
									: 'text-muted-foreground hover:text-foreground'
							]}
							aria-pressed={option.value === followUpCondition}
							disabled={actionsDisabled}
							onclick={() => builder.setFollowUpCondition(index, option.value)}
						>
							{option.label()}
						</button>
					{/each}
				</div>
				{#if followUpCondition === 'always'}
					<p class="text-xs text-muted-foreground">
						{m.follow_up_always_helper()}
					</p>
				{/if}
			</div>
		{/if}
	{/snippet}
</StepCardDisplay>
