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
	import { sameUnit, type ActiveUnit } from './scroll-follow/active-unit.js';
	import { createPeerScrollFollow } from './scroll-follow/peer-scroll-follow.js';
	import {
		findNearestUnitToLine,
		findRangeForUnit,
		findUnitAtLine,
		mapYamlCardRanges,
		type YamlCardRange
	} from './scroll-follow/yaml-ranges.js';

	//

	let { self: builder }: SelfProp<StepsBuilder> = $props();

	const { debugEntityData } = steps;

	let addStepPane: PaneHandle | null = $state(null);
	let stepsPane: PaneHandle | null = $state(null);
	let rightPane: PaneHandle | null = $state(null);
	let cardsScrollContainer: HTMLElement | null = $state(null);
	/** Half-viewport end pad so the last (short) card can scroll to center. */
	let cardsEndPadPx = $state(0);
	let yamlScroller: HTMLElement | null = $state(null);

	const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
	const editingIndex = $derived(formMode?.intent === 'edit' ? formMode.stepIndex : undefined);
	const columnTitle = $derived(formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step());
	const stepDocsUrl = $derived(formMode?.config.docsUrl);
	const rightColumnTitle = $derived(builder.isManualMode ? m.manual_edit() : m.YAML_preview());

	let lastAppliedManualMode: boolean | null = $state(null);

	const peerScroll = createPeerScrollFollow();
	let scrollFollowEnabled = $state(peerScroll.enabled);
	/** Explicit user selection (YAML click). Edit focus is derived separately. */
	let pinnedSelectedUnit = $state<ActiveUnit | null>(null);
	/** Pointer hover from a step card or a YAML step range (independent of selection). */
	let hoveredUnit = $state<ActiveUnit | null>(null);

	$effect(() => {
		const sync = () => {
			scrollFollowEnabled = peerScroll.enabled;
		};
		sync();
		return peerScroll.subscribe(sync);
	});

	$effect(() => () => peerScroll.dispose());

	const yamlRanges = $derived(
		builder.isManualMode || String.isEmpty(builder.yamlPreview)
			? ([] as YamlCardRange[])
			: mapYamlCardRanges(builder.yamlPreview)
	);

	/** Selected: edit focus or explicit YAML click — never scroll-follow activeUnit. */
	const selectedUnit = $derived.by((): ActiveUnit | null => {
		if (builder.isManualMode) return null;
		if (editingIndex !== undefined) return { section: 'steps', index: editingIndex };
		return pinnedSelectedUnit;
	});

	const selectedLines = $derived.by(() => {
		const unit = selectedUnit;
		if (!unit) return null;
		const range = findRangeForUnit(yamlRanges, unit.section, unit.index);
		if (!range) return null;
		return { start: range.startLine, end: range.endLine };
	});

	const hoverLines = $derived.by(() => {
		const unit = hoveredUnit;
		if (!unit) return null;
		const range = findRangeForUnit(yamlRanges, unit.section, unit.index);
		if (!range) return null;
		return { start: range.startLine, end: range.endLine };
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

	// Edit selection drives wash; with scroll-follow also smooth-scroll YAML.
	$effect(() => {
		if (builder.isManualMode) return;
		peerScroll.onEditFocus(editingIndex);
	});

	// Keep end pad ~half the cards scrollport so last short cards can center.
	$effect(() => {
		const el = cardsScrollContainer;
		if (!el) {
			cardsEndPadPx = 0;
			return;
		}
		const update = () => {
			cardsEndPadPx = Math.round(el.clientHeight * 0.3);
		};
		update();
		const ro = new ResizeObserver(update);
		ro.observe(el);
		return () => ro.disconnect();
	});

	// After add/clone, scroll the new card into view (independent of selection highlight).
	$effect(() => {
		const index = builder.revealStepIndex;
		if (index == null || builder.isManualMode) return;
		const cardsEl = cardsScrollContainer;
		// Wait until the steps scroller exists (first card mount).
		if (!cardsEl) return;
		builder.revealStepIndex = null;
		const unit: ActiveUnit = { section: 'steps', index };
		void tick().then(() => {
			peerScroll.onReveal(unit);
		});
	});

	// Debounced re-follow when YAML text regenerates (not when activeUnit changes —
	// that dependency was yanking YAML to start-band ~130ms after each step change).
	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const yaml = builder.yamlPreview;
		const ranges = yamlRanges;
		if (!yaml || ranges.length === 0) return;
		return peerScroll.onYamlTextChanged();
	});

	// Bind whenever not manual so onReveal can center-scroll even with follow off.
	// Handlers no-op while disabled.
	$effect(() => {
		if (builder.isManualMode) return;
		const cardsEl = cardsScrollContainer;
		if (!cardsEl) return;
		return peerScroll.bindCards(cardsEl);
	});

	$effect(() => {
		if (builder.isManualMode) return;
		const yamlEl = yamlScroller;
		if (!yamlEl) return;
		// getRanges reads the live derived — do not rebind on every YAML regen.
		return peerScroll.bindYaml(yamlEl, () => yamlRanges);
	});

	function setScrollFollowEnabled(checked: boolean) {
		peerScroll.setEnabled(checked, editingIndex);
	}

	function onYamlLineClick(line: number) {
		const hit = findUnitAtLine(yamlRanges, line);
		if (!hit) return;
		const next: ActiveUnit = { section: hit.section, index: hit.index };
		pinnedSelectedUnit = next;
		peerScroll.followUnit(next, 'yaml');
	}

	function onYamlLineHover(line: number | null) {
		if (line === null) {
			hoveredUnit = null;
			return;
		}
		const hit = findNearestUnitToLine(yamlRanges, line);
		if (!hit) return;
		const next: ActiveUnit = { section: hit.section, index: hit.index };
		if (sameUnit(hoveredUnit, next)) return;
		hoveredUnit = next;
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
		bind:scrollContainer={cardsScrollContainer}
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
								hoveredUnit = { section: 'steps', index };
							}}
							onmouseleave={() => {
								if (
									hoveredUnit?.section === 'steps' &&
									hoveredUnit.index === index
								) {
									hoveredUnit = null;
								}
							}}
						>
							<StepCard
								{builder}
								{step}
								{index}
								editing={editingIndex === index}
								selected={selectedUnit?.section === 'steps' &&
									selectedUnit.index === index}
								hovered={hoveredUnit?.section === 'steps' &&
									hoveredUnit.index === index}
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
							checked={scrollFollowEnabled}
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
		{:else if String.isEmpty(builder.yamlPreview)}
			<EmptyState text={m.YAML_preview_will_appear_here()} />
		{:else}
			<CodeDisplay
				content={builder.yamlPreview}
				language="yaml"
				containerClass="rounded-none h-full min-h-0 grow"
				contentClass="text-sm"
				{selectedLines}
				{hoverLines}
				endPadRatio={0.3}
				onLineClick={onYamlLineClick}
				onLineHover={onYamlLineHover}
				bind:scroller={yamlScroller}
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
