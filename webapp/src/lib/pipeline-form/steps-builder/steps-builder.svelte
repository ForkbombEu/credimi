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
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import { Switch } from '@/components/ui/switch';
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
	import {
		resolveViewportActiveCard,
		resolveViewportYamlLine,
		sameUnit,
		scrollCardIntoView,
		scrollYamlLineIntoView,
		type ActiveUnit
	} from './scroll-follow/active-unit.js';
	import {
		readScrollFollowEnabled,
		writeScrollFollowEnabled
	} from './scroll-follow/preference.js';
	import {
		findRangeForUnit,
		findUnitAtLine,
		firstStepStartLine,
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
	let yamlScroller: HTMLElement | null = $state(null);

	const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
	const editingIndex = $derived(formMode?.intent === 'edit' ? formMode.stepIndex : undefined);
	const columnTitle = $derived(formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step());
	const stepDocsUrl = $derived(formMode?.config.docsUrl);
	const rightColumnTitle = $derived(builder.isManualMode ? m.manual_edit() : m.YAML_preview());

	let lastAppliedManualMode: boolean | null = $state(null);

	let scrollFollowEnabled = $state(readScrollFollowEnabled());
	let activeUnit = $state<ActiveUnit | null>(null);
	let scrollLeader: 'cards' | 'yaml' | null = null;
	let scrollLockUntil = 0;

	const LOCK_MS = 220;
	const YAML_REGEN_DEBOUNCE_MS = 130;

	const yamlRanges = $derived(
		builder.isManualMode || String.isEmpty(builder.yamlPreview)
			? ([] as YamlCardRange[])
			: mapYamlCardRanges(builder.yamlPreview)
	);

	const highlightLines = $derived.by(() => {
		if (!scrollFollowEnabled || !activeUnit) return null;
		const range = findRangeForUnit(yamlRanges, activeUnit.section, activeUnit.index);
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

	// Edit selection drives active unit (instant YAML follow)
	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		if (editingIndex === undefined) return;
		const next: ActiveUnit = { section: 'steps', index: editingIndex };
		if (sameUnit(activeUnit, next)) return;
		activeUnit = next;
		followPeerFromCards('auto');
	});

	// Debounced re-follow when YAML text regenerates
	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const yaml = builder.yamlPreview;
		const unit = activeUnit;
		const ranges = yamlRanges;
		if (!unit || !yaml || ranges.length === 0) return;

		const timer = setTimeout(() => {
			if (!sameUnit(activeUnit, unit)) return;
			followPeerFromCards('auto');
		}, YAML_REGEN_DEBOUNCE_MS);

		return () => clearTimeout(timer);
	});

	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const cardsEl = cardsScrollContainer;
		if (!cardsEl) return;

		const onScroll = () => {
			if (Date.now() < scrollLockUntil && scrollLeader === 'yaml') return;
			scrollLeader = 'cards';
			scrollLockUntil = Date.now() + LOCK_MS;
			const next = resolveViewportActiveCard(cardsEl, activeUnit);
			if (!next || sameUnit(activeUnit, next)) return;
			activeUnit = next;
			followPeerFromCards('smooth');
		};

		cardsEl.addEventListener('scroll', onScroll, { passive: true });
		return () => cardsEl.removeEventListener('scroll', onScroll);
	});

	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const yamlEl = yamlScroller;
		if (!yamlEl) return;

		const onScroll = () => {
			if (Date.now() < scrollLockUntil && scrollLeader === 'cards') return;
			scrollLeader = 'yaml';
			scrollLockUntil = Date.now() + LOCK_MS;
			const firstStep = firstStepStartLine(yamlRanges);
			const line = resolveViewportYamlLine(yamlEl, firstStep);
			if (line === null) {
				activeUnit = null;
				return;
			}
			const hit = findUnitAtLine(yamlRanges, line);
			if (!hit) {
				activeUnit = null;
				return;
			}
			const next: ActiveUnit = { section: hit.section, index: hit.index };
			if (sameUnit(activeUnit, next)) return;
			activeUnit = next;
			followPeerFromYaml('smooth');
		};

		yamlEl.addEventListener('scroll', onScroll, { passive: true });
		return () => yamlEl.removeEventListener('scroll', onScroll);
	});

	function setScrollFollowEnabled(checked: boolean) {
		scrollFollowEnabled = checked;
		writeScrollFollowEnabled(checked);
		if (!checked) {
			activeUnit = null;
			return;
		}
		if (editingIndex !== undefined) {
			activeUnit = { section: 'steps', index: editingIndex };
			followPeerFromCards('auto');
		}
	}

	function followPeerFromCards(behavior: ScrollBehavior) {
		const unit = activeUnit;
		const yamlEl = yamlScroller;
		if (!unit || !yamlEl) return;
		const range = findRangeForUnit(yamlRanges, unit.section, unit.index);
		if (!range) return;
		scrollLeader = 'cards';
		scrollLockUntil = Date.now() + LOCK_MS;
		scrollYamlLineIntoView(yamlEl, range.startLine, behavior);
	}

	function followPeerFromYaml(behavior: ScrollBehavior) {
		const unit = activeUnit;
		const cardsEl = cardsScrollContainer;
		if (!unit || !cardsEl) return;
		scrollLeader = 'yaml';
		scrollLockUntil = Date.now() + LOCK_MS;
		scrollCardIntoView(cardsEl, unit, behavior);
	}

	function onYamlLineClick(line: number) {
		if (!scrollFollowEnabled) return;
		const hit = findUnitAtLine(yamlRanges, line);
		if (!hit) return;
		const next: ActiveUnit = { section: hit.section, index: hit.index };
		activeUnit = next;
		followPeerFromYaml('smooth');
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
			<div class="space-y-3 p-4">
				{#each builder.steps as step, index (step)}
					<div
						animate:flip={{ duration: 300 }}
						data-card-section="steps"
						data-card-index={index}
						tabindex="-1"
					>
						<StepCard {builder} {step} {index} editing={editingIndex === index} />
					</div>
				{/each}
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
			<div class="flex items-center gap-3">
				{#if !builder.isManualMode}
					<label
						class="flex items-center gap-2 text-xs font-medium text-muted-foreground"
					>
						<span>{m.Scroll_follow()}</span>
						<Switch
							checked={scrollFollowEnabled}
							onCheckedChange={setScrollFollowEnabled}
						/>
					</label>
				{/if}
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
				containerClass="rounded-none grow min-h-0"
				contentClass="text-sm"
				{highlightLines}
				onLineClick={scrollFollowEnabled ? onYamlLineClick : undefined}
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
