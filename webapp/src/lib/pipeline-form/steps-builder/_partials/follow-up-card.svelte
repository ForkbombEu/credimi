<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PipelineFinallyCondition } from '$lib/pipeline/types';
	import type { EnrichedFollowUp } from '$pipeline-form/functions.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { HelpCircle, PencilIcon, TrashIcon, XIcon } from '@lucide/svelte';
	import { Render } from '$lib/renderable';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import { InCardFormShell, isStepEditable, StepCardDisplay } from './index.js';
	import { useHeldFormMode } from './use-held-form-mode.svelte.js';

	//

	const DEFAULT_MAX_HEIGHT_PX = 480;

	type Props = {
		index: number;
		followUp: EnrichedFollowUp;
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
		maxHeightPx = DEFAULT_MAX_HEIGHT_PX
	}: Props = $props();

	const editable = $derived(isStepEditable(followUp.step));
	const actionsDisabled = $derived(builder.isFormMode);

	const held = useHeldFormMode(
		() => builder,
		() => editing
	);
	const showFormBody = $derived(held.mode !== null);
	let enterComplete = $state(false);

	const conditionOptions: { value: PipelineFinallyCondition; label: () => string }[] = [
		{ value: 'always', label: () => m.Always() },
		{ value: 'on_success', label: () => m.On_success() },
		{ value: 'on_failure', label: () => m.On_failure() }
	];
</script>

<div
	class={[
		'flex min-h-0 flex-col transition-opacity duration-200',
		showFormBody && enterComplete && 'h-full',
		faded && 'pointer-events-none opacity-40'
	]}
	style:max-height={showFormBody && enterComplete ? `${maxHeightPx}px` : undefined}
>
	<StepCardDisplay
		step={followUp.step}
		{editing}
		{selected}
		{hovered}
		{showFormBody}
		{expandReady}
		{maxHeightPx}
		bind:enterComplete
		class={showFormBody && enterComplete ? 'h-full' : undefined}
	>
		{#snippet topRight()}
			{#if showFormBody}
				<div class="flex items-center gap-1 pr-1">
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
				<div
					class={[
						'flex items-center gap-1 pr-1 transition-opacity',
						faded
							? 'opacity-100'
							: actionsDisabled
								? 'opacity-30'
								: 'opacity-30 group-hover:opacity-100'
					]}
				>
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
					onCollapsed={() => held.clear()}
				>
					{#snippet form()}
						<Render item={mode.form} />
					{/snippet}
				</InCardFormShell>
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
</div>
