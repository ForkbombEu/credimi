<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PipelineFinallyCondition } from '$lib/pipeline/types';
	import type { EnrichedFollowUp } from '$pipeline-form/functions.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { PencilIcon, TrashIcon } from '@lucide/svelte';
	import { Render } from '$lib/renderable';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import { isStepEditable, StepCardDisplay } from './index.js';
	import { useCardShell } from './use-card-shell.svelte.js';

	//

	type Props = {
		index: number;
		followUp: EnrichedFollowUp;
		builder: StepsBuilder;
		editing?: boolean;
		/** Enter-ready: parent `editing && session.inCard.phase === 'still'`. */
		expandReady?: boolean;
		selected?: boolean;
		hovered?: boolean;
		/** Sibling of the card being edited in place: dimmed and non-interactive (except the pencil). */
		faded?: boolean;
		/** Max height of the card while editing, in px. Composer passes `session.cardFillMaxPx`. */
		maxHeightPx?: number;
		/** Paired with held-form clear — Twin-pane `noteExitComplete`. */
		onExitUnlock?: () => void;
	};

	let {
		builder,
		followUp,
		index,
		editing = false,
		expandReady = false,
		selected = false,
		hovered = false,
		faded = false,
		maxHeightPx,
		onExitUnlock
	}: Props = $props();

	const editable = $derived(isStepEditable(followUp.step));
	const actionsDisabled = $derived(builder.isFormMode);

	const shell = useCardShell(
		() => builder,
		() => editing,
		() => {
			onExitUnlock?.();
		}
	);
	const showFormBody = $derived(shell.mode !== null);
	const canSave = $derived(Boolean(editing && shell.mode?.form.canSave()));

	const conditionOptions: { value: PipelineFinallyCondition; label: () => string }[] = [
		{ value: 'always', label: () => m.Always() },
		{ value: 'on_success', label: () => m.On_success() },
		{ value: 'on_failure', label: () => m.On_failure() }
	];
</script>

<StepCardDisplay
	step={followUp.step}
	{editing}
	{selected}
	{hovered}
	{faded}
	{showFormBody}
	{expandReady}
	{maxHeightPx}
	{actionsDisabled}
	docsUrl={shell.mode?.config.docsUrl}
	{canSave}
	onSave={() => {
		shell.mode?.form.commit();
	}}
	onDismiss={() => builder.exitFormState()}
	onExitComplete={() => {
		shell.completeExit();
	}}
>
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

	{#snippet formBody()}
		{#if shell.mode}
			<Render item={shell.mode.form} />
		{/if}
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
