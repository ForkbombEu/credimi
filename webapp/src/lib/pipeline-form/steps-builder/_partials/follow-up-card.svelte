<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PipelineFinallyCondition } from '$lib/pipeline/types';
	import type { EnrichedFollowUp } from '$pipeline-form/functions.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { PencilIcon, TrashIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import { isStepEditable, StepCardDisplay } from './index.js';

	//

	type Props = {
		index: number;
		followUp: EnrichedFollowUp;
		builder: StepsBuilder;
		editing?: boolean;
		selected?: boolean;
		hovered?: boolean;
	};

	let {
		builder,
		followUp,
		index,
		editing = false,
		selected = false,
		hovered = false
	}: Props = $props();

	const editable = $derived(isStepEditable(followUp.step));
	const actionsDisabled = $derived(builder.isFormMode);

	const conditionOptions: { value: PipelineFinallyCondition; label: () => string }[] = [
		{ value: 'always', label: () => m.Always() },
		{ value: 'on_success', label: () => m.On_success() },
		{ value: 'on_failure', label: () => m.On_failure() }
	];
</script>

<StepCardDisplay step={followUp.step} {editing} {selected} {hovered}>
	{#snippet topRight()}
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
	{/snippet}

	{#snippet footer()}
		<div class="flex flex-wrap items-center gap-x-2 gap-y-1 bg-slate-50 px-3 py-1.5">
			<div class="inline-flex rounded-md border bg-background p-0.5">
				{#each conditionOptions as option (option.value)}
					<button
						type="button"
						class={[
							'rounded-sm px-2 py-1 text-[11px] font-medium transition-colors',
							option.value === followUp.condition
								? 'bg-muted text-foreground'
								: 'text-muted-foreground hover:text-foreground'
						]}
						aria-pressed={option.value === followUp.condition}
						disabled={actionsDisabled}
						onclick={() => builder.setFollowUpCondition(index, option.value)}
					>
						{option.label()}
					</button>
				{/each}
			</div>
			{#if followUp.condition === 'always'}
				<p class="text-xs text-muted-foreground">
					{m.follow_up_always_helper()}
				</p>
			{/if}
		</div>
	{/snippet}
</StepCardDisplay>
