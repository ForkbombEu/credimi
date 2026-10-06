<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { ArrowDownIcon, ArrowUpIcon, CopyPlus, PencilIcon, TrashIcon } from '@lucide/svelte';
	import { comp, Render } from '$lib/renderable';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import ContinueOnErrorFooter from './continue-on-error-footer.svelte';
	import { isStepEditable, StepCardDisplay } from './index.js';
	import { useCardShell } from './use-card-shell.svelte.js';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
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
		onShift: (change: number) => void;
		/** Paired with held-form clear — Twin-pane `noteExitComplete`. */
		onExitUnlock?: () => void;
	};

	let {
		builder,
		step,
		index,
		editing = false,
		expandReady = false,
		selected = false,
		hovered = false,
		faded = false,
		maxHeightPx,
		onShift,
		onExitUnlock
	}: Props = $props();

	const editable = $derived(isStepEditable(step));
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
</script>

<StepCardDisplay
	{step}
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

	{#snippet formBody()}
		{#if shell.mode}
			<Render item={shell.mode.form} />
		{/if}
	{/snippet}
</StepCardDisplay>
