<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import {
		ArrowDownIcon,
		ArrowUpIcon,
		CopyPlus,
		HelpCircle,
		PencilIcon,
		TrashIcon,
		XIcon
	} from '@lucide/svelte';
	import { comp, Render } from '$lib/renderable';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import ContinueOnErrorFooter from './continue-on-error-footer.svelte';
	import { InCardFormShell, isStepEditable, StepCardDisplay } from './index.js';
	import { useHeldFormMode } from './use-held-form-mode.svelte.js';

	//

	const DEFAULT_MAX_HEIGHT_PX = 480;

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
		editing?: boolean;
		/** After enter scroll settles — drives the expand animation. */
		expandReady?: boolean;
		selected?: boolean;
		hovered?: boolean;
		/** Sibling of the card being edited in place: dimmed and non-interactive (except the pencil). */
		faded?: boolean;
		/** Max height of the card while editing, in px. */
		maxHeightPx?: number;
		onShift: (change: number) => void;
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
		maxHeightPx = DEFAULT_MAX_HEIGHT_PX,
		onShift
	}: Props = $props();

	const editable = $derived(isStepEditable(step));
	const actionsDisabled = $derived(builder.isFormMode);

	const held = useHeldFormMode(
		() => builder,
		() => editing
	);
	const showFormBody = $derived(held.mode !== null);
	let enterComplete = $state(false);
</script>

<div
	class={[
		'flex min-h-0 flex-col transition-opacity duration-200',
		faded && 'pointer-events-none opacity-40'
	]}
	style:max-height={showFormBody ? `${maxHeightPx}px` : undefined}
>
	<StepCardDisplay
		{step}
		{editing}
		{selected}
		{hovered}
		{showFormBody}
		{expandReady}
		{maxHeightPx}
		bind:enterComplete
		onExitComplete={() => held.clear()}
		footer={comp(ContinueOnErrorFooter, {
			step,
			onCheckedChange: (checked) => builder.setContinueOnError(index, checked)
		})}
	>
		{#snippet topRight()}
			{#if showFormBody}
				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="flex items-center gap-1 pr-1"
					onclick={(e) => e.stopPropagation()}
					onpointerdown={(e) => e.stopPropagation()}
				>
					{#if held.mode?.config.docsUrl}
						<IconButton
							variant="ghost"
							href={held.mode.config.docsUrl}
							target="_blank"
							rel="noopener noreferrer"
							icon={HelpCircle}
							size="xs"
							tooltip={m.Documentation()}
						/>
					{/if}
					<IconButton
						variant="ghost"
						icon={XIcon}
						size="xs"
						tooltip={m.Close()}
						onclick={() => builder.exitFormState()}
						data-testid="in-card-form-dismiss"
					/>
				</div>
			{:else}
				<!-- Prevent action chrome from selecting the card via bubbled click/pointerdown. -->
				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class={[
						'flex items-center gap-1 pr-1 transition-opacity',
						faded
							? 'opacity-100'
							: actionsDisabled
								? 'opacity-30'
								: 'opacity-30 group-hover:opacity-100'
					]}
					onclick={(e) => e.stopPropagation()}
					onpointerdown={(e) => e.stopPropagation()}
				>
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
				</div>
			{/if}
		{/snippet}

		{#snippet formBody()}
			{#if held.mode}
				{@const mode = held.mode}
				<InCardFormShell
					expanded={editing}
					canSave={editing && mode.form.canSave()}
					onSave={() => mode.form.commit()}
					onDismiss={() => builder.exitFormState()}
				>
					{#snippet form()}
						<Render item={mode.form} />
					{/snippet}
				</InCardFormShell>
			{/if}
		{/snippet}
	</StepCardDisplay>
</div>
