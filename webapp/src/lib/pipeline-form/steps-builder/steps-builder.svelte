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
	import {
		resolveViewportActiveCard,
		resolveViewportYamlLine,
		sameUnit,
		scrollCardIntoView,
		scrollYamlLineIntoView,
		watchDrivenScroll,
		type ActiveUnit
	} from './scroll-follow/active-unit.js';
	import {
		readScrollFollowEnabled,
		writeScrollFollowEnabled
	} from './scroll-follow/preference.js';
	import {
		findNearestUnitToLine,
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
	/** Half-viewport end pad so the last (short) card can scroll to center. */
	let cardsEndPadPx = $state(0);
	let yamlScroller: HTMLElement | null = $state(null);

	const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
	const editingIndex = $derived(formMode?.intent === 'edit' ? formMode.stepIndex : undefined);
	const columnTitle = $derived(formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step());
	const stepDocsUrl = $derived(formMode?.config.docsUrl);
	const rightColumnTitle = $derived(builder.isManualMode ? m.manual_edit() : m.YAML_preview());

	let lastAppliedManualMode: boolean | null = $state(null);

	let scrollFollowEnabled = $state(readScrollFollowEnabled());
	/** Viewport sync only — never drives wash/ring highlight. */
	let activeUnit = $state<ActiveUnit | null>(null);
	/** Explicit user selection (YAML click). Edit focus is derived separately. */
	let pinnedSelectedUnit = $state<ActiveUnit | null>(null);
	/** Pointer hover from a step card or a YAML step range (independent of selection). */
	let hoveredUnit = $state<ActiveUnit | null>(null);
	/** Side currently being scrolled by peer-follow; its scroll events must not reverse-drive. */
	let drivenSide: 'cards' | 'yaml' | null = null;
	let clearDriven: (() => void) | null = null;
	/**
	 * Side the user is actively scrolling. Held through an idle window so peer
	 * follow cannot reclaim leadership mid-gesture (which pushed YAML back).
	 */
	let scrollLeader: 'cards' | 'yaml' | null = null;
	/** Last side the user actually gestured on (wheel/touch/pointer). Survives leader idle
	 *  so momentum scroll can continue follow, while residual peer scrolls cannot steal. */
	let lastIntentSide: 'cards' | 'yaml' | null = null;
	let leaderIdleTimer: ReturnType<typeof setTimeout> | null = null;
	/** Coalesce peer follow to one update per frame (avoids stacked smooth/jank). */
	let peerFollowRaf: number | null = null;

	const YAML_REGEN_DEBOUNCE_MS = 130;
	const LEADER_IDLE_MS = 900;

	function claimScrollLeader(side: 'cards' | 'yaml') {
		scrollLeader = side;
		if (leaderIdleTimer) clearTimeout(leaderIdleTimer);
		leaderIdleTimer = setTimeout(() => {
			scrollLeader = null;
			leaderIdleTimer = null;
		}, LEADER_IDLE_MS);
	}

	function schedulePeerFollow(from: 'cards' | 'yaml') {
		if (peerFollowRaf != null) cancelAnimationFrame(peerFollowRaf);
		peerFollowRaf = requestAnimationFrame(() => {
			peerFollowRaf = null;
			// Smooth peer follow both ways; RAF coalesces to one scrollTo per frame.
			// Cards→YAML still uses start-band (see followPeerFromCards) so mid-gesture
			// only animates when the line leaves the upper zone.
			if (from === 'cards') followPeerFromCards('smooth');
			else followPeerFromYaml('smooth');
		});
	}

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
		if (editingIndex === undefined) return;
		const next: ActiveUnit = { section: 'steps', index: editingIndex };
		if (sameUnit(activeUnit, next)) return;
		activeUnit = next;
		if (scrollFollowEnabled) followPeerFromCards('smooth');
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
			scrollCardIntoView(cardsEl, unit, 'smooth', { align: 'center', focus: false });
			if (!scrollFollowEnabled) return;
			activeUnit = unit;
			followPeerFromCards('smooth');
		});
	});

	// Debounced re-follow when YAML text regenerates (not when activeUnit changes —
	// that dependency was yanking YAML to start-band ~130ms after each step change).
	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const yaml = builder.yamlPreview;
		const ranges = yamlRanges;
		if (!yaml || ranges.length === 0) return;

		const timer = setTimeout(() => {
			if (!activeUnit) return;
			followPeerFromCards('smooth');
		}, YAML_REGEN_DEBOUNCE_MS);

		return () => clearTimeout(timer);
	});

	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const cardsEl = cardsScrollContainer;
		if (!cardsEl) return;

		const onUserIntent = () => {
			if (drivenSide === 'cards') clearDriven?.();
			lastIntentSide = 'cards';
			claimScrollLeader('cards');
		};

		const onScroll = () => {
			if (drivenSide === 'cards') return;
			// Only the user-intent side may own scroll-follow. Residual peer scrolls after
			// leader idle used to claim here and yank YAML (start-band) mid-gesture.
			if (scrollLeader === 'yaml') return;
			if (scrollLeader !== 'cards' && lastIntentSide !== 'cards') return;
			claimScrollLeader('cards');
			const next = resolveViewportActiveCard(cardsEl, activeUnit);
			if (!next || sameUnit(activeUnit, next)) return;
			activeUnit = next;
			schedulePeerFollow('cards');
		};

		cardsEl.addEventListener('wheel', onUserIntent, { passive: true });
		cardsEl.addEventListener('touchstart', onUserIntent, { passive: true });
		cardsEl.addEventListener('pointerdown', onUserIntent, { passive: true });
		cardsEl.addEventListener('scroll', onScroll, { passive: true });
		return () => {
			cardsEl.removeEventListener('wheel', onUserIntent);
			cardsEl.removeEventListener('touchstart', onUserIntent);
			cardsEl.removeEventListener('pointerdown', onUserIntent);
			cardsEl.removeEventListener('scroll', onScroll);
		};
	});

	$effect(() => {
		if (!scrollFollowEnabled || builder.isManualMode) return;
		const yamlEl = yamlScroller;
		if (!yamlEl) return;

		const onUserIntent = () => {
			// User scrolling YAML cancels any peer-driven YAML animation and keeps leadership.
			if (drivenSide === 'yaml') clearDriven?.();
			lastIntentSide = 'yaml';
			claimScrollLeader('yaml');
		};

		const onScroll = () => {
			if (drivenSide === 'yaml') return;
			if (scrollLeader === 'cards') return;
			if (scrollLeader !== 'yaml' && lastIntentSide !== 'yaml') return;
			// Refresh leader on momentum scroll after idle so wash keeps tracking.
			claimScrollLeader('yaml');
			const firstStep = firstStepStartLine(yamlRanges);
			const line = resolveViewportYamlLine(yamlEl, firstStep);
			if (line === null) {
				activeUnit = null;
				return;
			}
			const hit = findNearestUnitToLine(yamlRanges, line);
			if (!hit) {
				activeUnit = null;
				return;
			}
			const next: ActiveUnit = { section: hit.section, index: hit.index };
			if (sameUnit(activeUnit, next)) return;
			activeUnit = next;
			schedulePeerFollow('yaml');
		};

		yamlEl.addEventListener('wheel', onUserIntent, { passive: true });
		yamlEl.addEventListener('touchstart', onUserIntent, { passive: true });
		yamlEl.addEventListener('pointerdown', onUserIntent, { passive: true });
		yamlEl.addEventListener('scroll', onScroll, { passive: true });
		return () => {
			yamlEl.removeEventListener('wheel', onUserIntent);
			yamlEl.removeEventListener('touchstart', onUserIntent);
			yamlEl.removeEventListener('pointerdown', onUserIntent);
			yamlEl.removeEventListener('scroll', onScroll);
		};
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
			followPeerFromCards('smooth');
		}
	}

	function beginDriven(side: 'cards' | 'yaml', el: HTMLElement, behavior: ScrollBehavior) {
		clearDriven?.();
		drivenSide = side;
		// Peer animation is on behalf of the opposite side — keep that side as leader
		// through the animation and a short idle after scrollend.
		const leader: 'cards' | 'yaml' = side === 'cards' ? 'yaml' : 'cards';
		claimScrollLeader(leader);
		clearDriven = watchDrivenScroll(el, behavior, () => {
			if (drivenSide === side) drivenSide = null;
			clearDriven = null;
			claimScrollLeader(leader);
		});
	}

	function followPeerFromCards(behavior: ScrollBehavior) {
		// Never drive YAML while the user owns that scroller (regen/edit effects used
		// to start-band yank mid YAML gesture).
		if (scrollLeader === 'yaml') return;
		const unit = activeUnit;
		const yamlEl = yamlScroller;
		if (!unit || !yamlEl) return;
		const range = findRangeForUnit(yamlRanges, unit.section, unit.index);
		if (!range) return;
		beginDriven('yaml', yamlEl, behavior);
		// Cards-led mid-gesture: start-band (only scroll when the line leaves the upper
		// zone). Otherwise hard start-align (edit selection, enable toggle, regen).
		const align = scrollLeader === 'cards' ? 'start-band' : 'start';
		if (!scrollYamlLineIntoView(yamlEl, range.startLine, behavior, align)) {
			clearDriven?.();
		}
	}

	function followPeerFromYaml(behavior: ScrollBehavior) {
		const unit = activeUnit;
		const cardsEl = cardsScrollContainer;
		if (!unit || !cardsEl) return;
		beginDriven('cards', cardsEl, behavior);
		// Mid-gesture: center each active card. nearest/start no-op when the new card is
		// already near the top while the previous card still owns the viewport center —
		// then the next off-screen unit leaps over the skipped visual step.
		const align = 'center';
		if (
			!scrollCardIntoView(cardsEl, unit, behavior, {
				align,
				focus: behavior === 'smooth'
			})
		) {
			clearDriven?.();
		}
	}

	function onYamlLineClick(line: number) {
		const hit = findUnitAtLine(yamlRanges, line);
		if (!hit) return;
		const next: ActiveUnit = { section: hit.section, index: hit.index };
		pinnedSelectedUnit = next;
		activeUnit = next;
		if (!scrollFollowEnabled) return;
		lastIntentSide = 'yaml';
		claimScrollLeader('yaml');
		followPeerFromYaml('smooth');
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
