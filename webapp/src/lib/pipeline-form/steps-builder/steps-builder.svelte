<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EntityData } from '$lib/global/entities.js';

	import {
		BlocksIcon,
		EllipsisIcon,
		HelpCircle,
		PencilIcon,
		RefreshCcwIcon,
		XIcon
	} from '@lucide/svelte';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { Render, type SelfProp } from '$lib/renderable';
	import * as steps from '$pipeline-form/steps';
	import { String } from 'effect';
	import { tick } from 'svelte';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import { m } from '@/i18n';

	import type { StepsBuilder } from './steps-builder.svelte.js';

	import {
		BulkWalletVersionChange,
		Column,
		EmptyState,
		ManualEditorColumn,
		StepCard
	} from './_partials/index.js';
	import { STEPS_BUILDER_PANE_LAYOUT as LAYOUT, type PaneHandle } from './pane-layout.js';

	//

	let { self: builder }: SelfProp<StepsBuilder> = $props();

	const { debugEntityData } = steps;

	let addStepPane: PaneHandle | null = $state(null);
	let stepsPane: PaneHandle | null = $state(null);
	let rightPane: PaneHandle | null = $state(null);

	const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
	const editingSection = $derived(
		formMode?.intent === 'edit' ? (formMode.section ?? 'steps') : undefined
	);
	const editingIndex = $derived(formMode?.intent === 'edit' ? formMode.stepIndex : undefined);
	const columnTitle = $derived(formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step());
	const stepDocsUrl = $derived(formMode?.config.docsUrl);
	const showFollowUpAddActions = $derived(builder.isFollowUpEligibleForm());
	const rightColumnTitle = $derived(builder.isManualMode ? m.manual_edit() : m.YAML_preview());

	let lastAppliedManualMode: boolean | null = null;
	let lastFocusedCardToken = 0;

	$effect(() => {
		const isManual = builder.isManualMode;
		if (!addStepPane || !stepsPane || !rightPane) return;
		if (lastAppliedManualMode === isManual) return;

		lastAppliedManualMode = isManual;
		if (isManual) {
			const layout = LAYOUT.manual;
			addStepPane.resize(layout.addStep);
			stepsPane.resize(layout.stepsSequence);
			rightPane.resize(layout.editor);
		} else {
			const layout = LAYOUT.blocks;
			addStepPane.resize(layout.addStep);
			stepsPane.resize(layout.stepsSequence);
			rightPane.resize(layout.right);
		}
	});

	$effect(() => {
		const createdCard = builder.createdCard;
		if (!createdCard || createdCard.token === lastFocusedCardToken) return;

		lastFocusedCardToken = createdCard.token;
		void tick().then(() => {
			const selector = `[data-card-section="${createdCard.section}"][data-card-index="${createdCard.index}"]`;
			const card = document.querySelector<HTMLElement>(selector);
			card?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
			card?.focus({ preventScroll: true });
		});
	});
</script>

<Resizable.PaneGroup direction="horizontal" class="min-h-0 grow gap-2">
	<Column
		bind:pane={addStepPane}
		title={columnTitle}
		defaultSize={LAYOUT.blocks.addStep}
		order={1}
		disabled={builder.isManualMode}
	>
		{#if builder.mode.id == 'form'}
			<div class="flex grow flex-col" in:fly>
				<Render item={builder.mode.form} />
				{#if formMode?.intent === 'edit'}
					<div class="mt-auto border-t p-4">
						<Button
							class="w-full"
							disabled={!formMode.form.canSave()}
							onclick={() => formMode.form.commit()}
						>
							{m.Save()}
						</Button>
					</div>
				{:else if showFollowUpAddActions && formMode}
					<div class="mt-auto space-y-2 border-t p-4">
						<Button
							class="w-full"
							disabled={!formMode.form.canSave()}
							onclick={() => formMode.form.commit()}
						>
							{m.Add_step()}
						</Button>
						<Button
							variant="outline"
							class="w-full"
							disabled={!formMode.form.canSave()}
							onclick={() => builder.addAsFollowUp()}
						>
							{m.Add_as_follow_up()}
						</Button>
					</div>
				{/if}
			</div>
		{:else}
			{@render stepButtons()}
		{/if}

		{#snippet titleRight()}
			{#if builder.mode.id == 'form'}
				<div class="flex items-center gap-1">
					{#if stepDocsUrl}
						<IconButton
							variant="outline"
							href={stepDocsUrl}
							target="_blank"
							rel="noopener noreferrer"
							icon={HelpCircle}
							size="xs"
							tooltip={m.Documentation()}
						/>
					{/if}
					<IconButton
						variant="outline"
						onclick={() => builder.exitFormState()}
						icon={XIcon}
						size="xs"
					/>
				</div>
			{/if}
		{/snippet}
	</Column>

	<Resizable.Handle class="hover:bg-primary" />

	<Column
		bind:pane={stepsPane}
		title={m.Steps_sequence()}
		defaultSize={LAYOUT.blocks.stepsSequence}
		order={2}
		disabled={builder.isManualMode}
	>
		{#snippet titleRight()}
			{#if !builder.isManualMode && builder.canOfferChangeWalletVersion()}
				<DropdownMenu
					items={[
						{
							label: m.Change_wallet_version(),
							onclick: () => builder.openChangeWalletVersion(),
							icon: RefreshCcwIcon
						}
					]}
				>
					{#snippet trigger({ props })}
						<IconButton {...props} icon={EllipsisIcon} size="xs" variant="ghost" />
					{/snippet}
				</DropdownMenu>
			{/if}
		{/snippet}

		<div class="space-y-4 p-4">
			{#if builder.steps.length > 0}
				<div class="space-y-3">
					{#each builder.steps as step, index (step)}
						<div
							animate:flip={{ duration: 300 }}
							data-card-section="steps"
							data-card-index={index}
							tabindex="-1"
						>
							<StepCard
								{builder}
								{step}
								{index}
								editing={editingSection === 'steps' && editingIndex === index}
							/>
						</div>
					{/each}
				</div>
			{:else if builder.isSavedManualPipeline}
				<EmptyState text={m.pipeline_manually_saved_no_cards()} />
			{:else}
				<EmptyState text={m.Pipeline_steps_will_appear_here()} />
			{/if}

			<div class="border-t pt-4">
				<h3 class="text-sm font-medium">{m.Follow_ups()}</h3>
			</div>

			{#if builder.followUps.length > 0}
				<div class="space-y-3">
					{#each builder.followUps as followUp, index (followUp)}
						<div
							animate:flip={{ duration: 300 }}
							data-card-section="follow-ups"
							data-card-index={index}
							tabindex="-1"
						>
							<StepCard
								{builder}
								step={followUp.step}
								{index}
								followUpCondition={followUp.condition}
								editing={editingSection === 'follow-ups' && editingIndex === index}
							/>
						</div>
					{/each}
				</div>
			{:else}
				<EmptyState text={m.no_follow_ups_hint()} />
			{/if}
		</div>
	</Column>

	<Resizable.Handle class="hover:bg-primary" />

	<Column
		bind:pane={rightPane}
		title={rightColumnTitle}
		class="card min-w-0 overflow-hidden"
		contentClass={builder.isManualMode ? 'overflow-hidden' : undefined}
		defaultSize={LAYOUT.blocks.right}
		order={3}
	>
		{#snippet titleRight()}
			{#if builder.mode.id === 'manual' && !builder.isManualLocked}
				<Button
					variant="link"
					class="h-fit gap-1 p-0 text-xs"
					onclick={() => void builder.exitManualMode()}
				>
					<BlocksIcon size={10} />
					{m.back_to_steps()}
				</Button>
			{:else if !builder.isManualMode}
				<Button
					variant="link"
					class="h-fit gap-1 p-0 text-xs"
					onclick={() => builder.enterManualMode(builder.yamlPreview)}
				>
					<PencilIcon size={10} />
					{m.edit_manually()}
				</Button>
			{/if}
		{/snippet}

		{#if builder.mode.id === 'manual'}
			<ManualEditorColumn editor={builder.mode.editor} />
		{:else if String.isEmpty(builder.yamlPreview)}
			<EmptyState text={m.YAML_preview_will_appear_here()} />
		{:else}
			<CodeDisplay
				content={builder.yamlPreview}
				language="yaml"
				containerClass="rounded-none grow"
				contentClass="text-sm"
			/>
		{/if}
	</Column>
</Resizable.PaneGroup>

<BulkWalletVersionChange {builder} bind:open={builder.changeWalletVersionDialogOpen} />

<!--  -->

{#snippet stepButtons()}
	<div class="flex flex-col gap-2 p-4" in:fly>
		{#each steps.coreConfigs as config (config.use)}
			{@render stepButton(config)}
		{/each}

		<div
			class="-mb-1 pt-2 text-[10px] font-medium tracking-normal text-muted-foreground uppercase"
		>
			{m.utils()}
		</div>

		{@render baseStepButton(debugEntityData, () => builder.addDebugStep())}

		{#each steps.utilsConfigs as config (config.use)}
			{@render stepButton(config)}
		{/each}
	</div>
{/snippet}

{#snippet stepButton(config: steps.AnyConfig)}
	{@render baseStepButton(steps.getDisplayData(config.use), () =>
		builder.initAddStep(config.use)
	)}
{/snippet}

{#snippet baseStepButton(displayData: EntityData, onClick: () => void)}
	<Button variant="outline" class="justify-start!" onclick={onClick}>
		<Icon src={displayData.icon} class={displayData.classes.text} />
		<span class="truncate">
			{displayData.labels.singular}
		</span>
	</Button>
{/snippet}
