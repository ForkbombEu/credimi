<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EntityData } from '$lib/global/entities.js';
	import type { Attachment } from 'svelte/attachments';

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
	import { String as EffectString } from 'effect';
	import { tick } from 'svelte';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import Switch from '@/components/ui/switch/switch.svelte';
	import { m } from '@/i18n';

	import type { StepsBuilder } from './steps-builder.svelte.js';

	import {
		BulkWalletVersionChange,
		Column,
		EmptyState,
		ManualEditorColumn,
		StepCard
	} from './_partials/index.js';
	import {
		applyStepsBuilderPaneLayout,
		STEPS_BUILDER_PANE_LAYOUT as LAYOUT,
		type PaneHandle
	} from './pane-layout.js';
	import { type ActiveUnit } from './scroll-follow/active-unit.js';
	import { PeerScrollFollow } from './scroll-follow/peer-scroll-follow.svelte.js';
	import { UnitHighlight } from './scroll-follow/unit-highlight.svelte.js';
	import { mapYamlCardRanges, type YamlCardRange } from './scroll-follow/yaml-ranges.js';

	//

	let { self: builder }: SelfProp<StepsBuilder> = $props();

	const { debugEntityData } = steps;

	let addStepPane: PaneHandle | null = $state(null);
	let stepsPane: PaneHandle | null = $state(null);
	let rightPane: PaneHandle | null = $state(null);
	/** Half-viewport end pad so the last (short) card can scroll to center. */
	let cardsEndPadPx = $state(0);

	function composeAttachments(...parts: Array<Attachment | undefined>): Attachment | undefined {
		const active = parts.filter((part): part is Attachment => part != null);
		if (active.length === 0) return undefined;
		if (active.length === 1) return active[0];
		return (node) => {
			const cleanups = active
				.map((attach) => attach(node))
				.filter((cleanup): cleanup is () => void => typeof cleanup === 'function');
			if (cleanups.length === 0) return;
			return () => {
				for (const cleanup of cleanups) cleanup();
			};
		};
	}

	const cardsEndPadAttach: Attachment = (el) => {
		const update = () => {
			cardsEndPadPx = Math.round(el.clientHeight * 0.3);
		};
		update();
		const ro = new ResizeObserver(update);
		ro.observe(el);
		return () => {
			ro.disconnect();
			cardsEndPadPx = 0;
		};
	};

	const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
	const editingIndex = $derived(formMode?.intent === 'edit' ? formMode.stepIndex : undefined);
	const columnTitle = $derived(formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step());
	const stepDocsUrl = $derived(formMode?.config.docsUrl);
	const rightColumnTitle = $derived(builder.isManualMode ? m.manual_edit() : m.YAML_preview());

	let lastAppliedManualMode: boolean | null = $state(null);

	const EMPTY_YAML_RANGES: YamlCardRange[] = [];
	const yamlRanges = $derived(
		builder.isManualMode || EffectString.isEmpty(builder.yamlPreview)
			? EMPTY_YAML_RANGES
			: mapYamlCardRanges(builder.yamlPreview)
	);

	const peerScroll = new PeerScrollFollow();
	const unitHighlight = new UnitHighlight({
		getIsManual: () => builder.isManualMode,
		getEditingIndex: () => editingIndex,
		getRanges: () => yamlRanges
	});

	builder.bindComposerScroll({
		onRevealStep: (index) => {
			if (builder.isManualMode) return;
			const unit: ActiveUnit = { section: 'steps', index };
			// Wait for the new card to mount before scrolling.
			void tick().then(() => peerScroll.onReveal(unit));
		},
		onEditFocus: (stepIndex) => {
			if (builder.isManualMode) return;
			peerScroll.onEditFocus(stepIndex);
		}
	});

	// Stable yaml attach — getter reads live ranges; do not recreate on every yaml regen.
	const yamlPreviewEmpty = $derived(EffectString.isEmpty(builder.yamlPreview));
	const yamlScrollAttach = $derived(
		builder.isManualMode || yamlPreviewEmpty
			? undefined
			: peerScroll.yamlAttach(() => yamlRanges)
	);

	// Compose peer-scroll + end-pad; identity stable unless isManualMode flips.
	const cardsScrollAttach = $derived(
		composeAttachments(
			!builder.isManualMode ? peerScroll.cardsAttach : undefined,
			cardsEndPadAttach
		)
	);

	$effect(() => () => {
		builder.bindComposerScroll({});
		peerScroll.dispose();
		unitHighlight.dispose();
	});

	$effect(() => {
		const isManual = builder.isManualMode;
		const panesReady = addStepPane && stepsPane && rightPane;
		if (!panesReady) return;
		if (lastAppliedManualMode === isManual) return;

		lastAppliedManualMode = isManual;
		applyStepsBuilderPaneLayout(
			{ addStep: addStepPane, stepsSequence: stepsPane, right: rightPane },
			isManual
		);
	});

	// Debounced re-follow when YAML text regenerates (not when activeUnit changes —
	// that dependency was yanking YAML to start-band ~130ms after each step change).
	$effect(() => {
		if (!peerScroll.enabled || builder.isManualMode) return;
		const yaml = builder.yamlPreview;
		const ranges = yamlRanges;
		if (!yaml || ranges.length === 0) return;
		return peerScroll.onYamlTextChanged();
	});

	function setScrollFollowEnabled(checked: boolean) {
		peerScroll.setEnabled(checked, editingIndex);
	}

	function onYamlLineClick(line: number) {
		const pinned = unitHighlight.pinYamlLine(line);
		if (!pinned) return;
		peerScroll.followUnit(pinned, 'yaml');
	}

	function onYamlLineHover(line: number | null) {
		unitHighlight.hoverYamlLine(line);
	}
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
		scrollAttach={cardsScrollAttach}
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

		{#if builder.steps.length > 0}
			<div class="flex flex-col p-4">
				<div class="space-y-3">
					{#each builder.steps as step, index (step)}
						<div
							animate:flip={{ duration: 300 }}
							data-card-section="steps"
							data-card-index={index}
							role="group"
							tabindex="-1"
							onmouseenter={() => {
								unitHighlight.hoverCard({ section: 'steps', index });
							}}
							onmouseleave={() => {
								unitHighlight.clearHoverCard({ section: 'steps', index });
							}}
						>
							<StepCard
								{builder}
								{step}
								{index}
								editing={editingIndex === index}
								selected={unitHighlight.isCardSelected('steps', index)}
								hovered={unitHighlight.isCardHovered('steps', index)}
							/>
						</div>
					{/each}
				</div>
				<!-- Lets peer-follow center the last short card (e.g. debug). -->
				<div
					class="pointer-events-none shrink-0"
					style:height="{cardsEndPadPx}px"
					aria-hidden="true"
				></div>
			</div>
		{:else if builder.isSavedManualPipeline}
			<EmptyState text={m.pipeline_manually_saved_no_cards()} />
		{:else}
			<EmptyState text={m.Pipeline_steps_will_appear_here()} />
		{/if}
	</Column>

	<Resizable.Handle class="hover:bg-primary" />

	<Column
		bind:pane={rightPane}
		title={rightColumnTitle}
		class="card min-w-0 overflow-hidden"
		contentClass="overflow-hidden"
		defaultSize={LAYOUT.blocks.right}
		order={3}
	>
		{#snippet titleRight()}
			<div class="flex min-w-0 items-center gap-1">
				{#if !builder.isManualMode}
					<label
						class="flex min-w-0 shrink items-center gap-0.5 text-xs font-medium text-primary hover:cursor-pointer hover:underline"
						title={m.Scroll_follow()}
					>
						<Switch
							checked={peerScroll.enabled}
							onCheckedChange={setScrollFollowEnabled}
							class="shrink-0 scale-75"
						/>
						<span class="truncate">{m.Scroll_follow()}</span>
					</label>
				{/if}
				{#if builder.mode.id === 'manual' && !builder.isManualLocked}
					<Button
						variant="link"
						class="h-fit min-w-0 shrink gap-1 p-0 text-xs has-[>svg]:px-0"
						title={m.back_to_steps()}
						onclick={() => void builder.exitManualMode()}
					>
						<BlocksIcon size={10} class="shrink-0" />
						<span class="truncate">{m.back_to_steps()}</span>
					</Button>
				{:else if !builder.isManualMode}
					<Button
						variant="link"
						class="h-fit min-w-0 shrink gap-1 p-0 text-xs has-[>svg]:px-0"
						title={m.edit_manually()}
						onclick={() => builder.enterManualMode(builder.yamlPreview)}
					>
						<PencilIcon size={10} class="shrink-0" />
						<span class="truncate">{m.edit_manually()}</span>
					</Button>
				{/if}
			</div>
		{/snippet}

		{#if builder.mode.id === 'manual'}
			<ManualEditorColumn editor={builder.mode.editor} />
		{:else if EffectString.isEmpty(builder.yamlPreview)}
			<EmptyState text={m.YAML_preview_will_appear_here()} />
		{:else}
			<CodeDisplay
				content={builder.yamlPreview}
				language="yaml"
				containerClass="rounded-none h-full min-h-0 grow"
				contentClass="text-sm"
				selectedLines={unitHighlight.selectedLines}
				hoverLines={unitHighlight.hoverLines}
				endPadRatio={0.3}
				onLineClick={onYamlLineClick}
				onLineHover={onYamlLineHover}
				scrollerAttach={yamlScrollAttach}
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
