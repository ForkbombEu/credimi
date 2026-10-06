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

	import type { InCardHostChrome } from './in-card-host-chrome.js';

	import InCardHost from './in-card-host.svelte';
	import { isStepEditable } from './utils.js';

	type Props = InCardHostChrome & {
		index: number;
		followUp: EnrichedFollowUp;
		builder: StepsBuilder;
	};

	let { builder, followUp, index, ...chrome }: Props = $props();

	const editable = $derived(isStepEditable(followUp.step));
	const actionsDisabled = $derived(builder.isFormMode);

	const conditionOptions: { value: PipelineFinallyCondition; label: () => string }[] = [
		{ value: 'always', label: () => m.Always() },
		{ value: 'on_success', label: () => m.On_success() },
		{ value: 'on_failure', label: () => m.On_failure() }
	];
</script>

<InCardHost {builder} step={followUp.step} {...chrome}>
	{#snippet topRight()}
		{#if editable}
			<IconButton
				icon={PencilIcon}
				variant="ghost"
				size="xs"
				class="pointer-events-auto"
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
</InCardHost>
