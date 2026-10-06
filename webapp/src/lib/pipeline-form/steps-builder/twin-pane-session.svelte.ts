// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Attachment } from 'svelte/attachments';

import { tick as svelteTick } from 'svelte';
import { get } from 'svelte/store';

import {
	createComposerVirtualizer,
	DEFAULT_STEP_ESTIMATE_SIZE,
	DEFAULT_YAML_STEP_ESTIMATE_SIZE,
	pinBothScrollports,
	restoreScrollTop as defaultRestoreScrollTop,
	stepCardSelector,
	yamlStepBlockSelector,
	type ComposerVirtualizer,
	type ComposerVirtualizerOptions
} from './composer-virtualizer.svelte.js';
import {
	createInCardSession,
	type InCardPhase,
	type InCardSession
} from './in-card-session.svelte.js';
import {
	createMultiListLayout,
	DEFAULT_YAML_LAYOUT_CHILDREN,
	type CardListLayout,
	type CreateCardListLayoutOptions,
	type MultiListLayoutEntry
} from './layout-swap.js';
import { runPairedShift as defaultRunPairedShift, type PairedShiftArgs } from './paired-shift.js';
import {
	ensureMountedForStepsVirtualizer,
	START_PADDING_PX,
	type ActiveUnit
} from './scroll-follow/active-unit.js';
import { nestedScrollerConsumesWheel } from './scroll-follow/nested-scroller-wheel.js';
import {
	PeerScrollFollow,
	type PeerScrollFollowOptions
} from './scroll-follow/peer-scroll-follow.svelte.js';
import { composeAttachments, endPadAttach } from './scroll-follow/scrollport-attachments.js';
import { UnitHighlight, type UnitHighlightInputs } from './scroll-follow/unit-highlight.svelte.js';

export type { InCardPhase };

export type TwinPaneCreatedCard = {
	section: 'steps' | 'follow-ups';
	index: number;
	token: number;
};

export type TwinPaneSessionOptions = {
	getStepsCount: () => number;
	getYamlStepsCount: () => number;
	getItemKey: (index: number) => string | number;
	getIsManual: () => boolean;
	getEditingIndex: () => number | undefined;
	getEditingSection: () => ActiveUnit['section'];
	getYamlPreview: () => string;
	getFollowUpsLength: () => number;
	getCreatedCard: () => TwinPaneCreatedCard | null;
	canShiftStep: (index: number, change: number) => boolean;
	mutateShiftStep: (index: number, change: number) => void;
	bindComposerScroll: (handlers: {
		onRevealStep?: (index: number) => void;
		onEditFocus?: (unit: ActiveUnit) => void;
	}) => void;
	/** Inject for tests. */
	createComposerVirtualizer?: (options: ComposerVirtualizerOptions) => ComposerVirtualizer;
	createPeerScrollFollow?: (options?: PeerScrollFollowOptions) => PeerScrollFollow;
	createUnitHighlight?: (inputs: UnitHighlightInputs) => UnitHighlight;
	createMultiListLayout?: (
		entries: MultiListLayoutEntry[],
		options?: CreateCardListLayoutOptions
	) => CardListLayout;
	runPairedShift?: (args: PairedShiftArgs) => Promise<void>;
	restoreScrollTop?: typeof defaultRestoreScrollTop;
	tick?: () => Promise<void>;
};

export type TwinPaneSession = {
	get cardsScroller(): HTMLElement | null;
	set cardsScroller(el: HTMLElement | null);
	get yamlScroller(): HTMLElement | null;
	set yamlScroller(el: HTMLElement | null);
	get stepsLayoutRoot(): HTMLElement | null;
	set stepsLayoutRoot(el: HTMLElement | null);
	get yamlStepsLayoutRoot(): HTMLElement | null;
	set yamlStepsLayoutRoot(el: HTMLElement | null);
	/**
	 * Cards scrollport end pad (view paints height). Idle: ~30% viewport from
	 * the observer. While In-card (aligning / still / exiting): at least
	 * viewport − start padding so the last card can start-align then expand.
	 */
	get cardsEndPadPx(): number;
	/** Cards scrollport clientHeight from the same ResizeObserver as the end pad. */
	get cardsViewportPx(): number;
	/**
	 * Whole-card max height while In-card filling the pane: at least 240px,
	 * otherwise viewport − start padding on both ends.
	 */
	get cardFillMaxPx(): number;
	/** Half-viewport end pad for the YAML scrollport (view paints height). */
	get yamlEndPadPx(): number;
	/** TanStack scrollMargin for YAML step blocks (header height inside scroller). */
	get yamlScrollMargin(): number;
	/** TanStack Readable — `$stepsVirt` in markup. */
	stepsVirt: ComposerVirtualizer['virtualizer'];
	/** TanStack Readable — `$yamlVirt` in markup (virtual items / totalSize). */
	yamlVirt: ComposerVirtualizer['virtualizer'];
	measureStepCard: Attachment;
	/** Measure YAML step blocks — mirrors `measureStepCard`. */
	measureYamlStep: Attachment;
	/**
	 * Peer-scroll + wheel lock + endPad for cards.
	 * Referentially stable for the session lifetime — `{@attach}` must not rebind
	 * when form mode goes idle (that would revert scrollTop to 0).
	 */
	get cardsScrollAttach(): Attachment | undefined;
	/**
	 * Peer-scroll + endPad for YAML. Referentially stable (handlers no-op when
	 * follow is off / preview empty); do not mint a new identity from mode.
	 */
	get yamlScrollAttach(): Attachment | undefined;
	/** YAML header → TanStack scrollMargin (view forwards from YamlPreviewPane). */
	setYamlHeaderHeight(height: number): void;
	isCardSelected(section: ActiveUnit['section'], index: number): boolean;
	isCardHovered(section: ActiveUnit['section'], index: number): boolean;
	hoverCard(unit: ActiveUnit): void;
	clearHoverCard(unit: ActiveUnit): void;
	get followEnabled(): boolean;
	shiftStep(index: number, change: number): Promise<void>;
	onUnitClick(unit: ActiveUnit, from: 'cards' | 'yaml'): void;
	onYamlUnitHover(unit: ActiveUnit | null): void;
	setFollowEnabled(checked: boolean): void;
	/**
	 * In-card still-ness phase machine (ADR 0012). Public: `phase` + `noteExitComplete`.
	 * Cards unlock via `inCard.noteExitComplete()` on exit-complete and onDestroy.
	 */
	readonly inCard: Pick<InCardSession, 'phase' | 'noteExitComplete'>;
	/**
	 * True while phase is `still` or `exiting` — Column scrollLocked + Switch disabled.
	 * Enter start-align (`aligning`) is not still yet.
	 */
	get still(): boolean;
	dispose(): void;
};

/**
 * Cards end-pad for In-card start-align. Idle keeps the observer's ~30% pad.
 * While In-card is active (`aligning` so enter can scroll, then `still` /
 * `exiting`), pad is at least `cardsViewportPx - START_PADDING_PX` so a last
 * (near-viewport-tall) card can sit at the pane start with room below.
 */
export function inCardCardsEndPadPx(
	observerPadPx: number,
	cardsViewportPx: number,
	inCardActive: boolean
): number {
	if (!inCardActive) return observerPadPx;
	return Math.max(observerPadPx, Math.max(0, cardsViewportPx - START_PADDING_PX));
}

/** Floor so a short pane still hosts a usable In-card form. */
export const MIN_CARD_FILL_MAX_PX = 240;

/**
 * Whole-card fill max (view paints `maxHeightPx`). Top + bottom inset matches
 * start-align padding so the open card fills the column with the same gap.
 */
export function inCardCardFillMaxPx(cardsViewportPx: number): number {
	return Math.max(MIN_CARD_FILL_MAX_PX, Math.max(0, cardsViewportPx - START_PADDING_PX * 2));
}

/**
 * Owns Pipeline Composer twin-pane lifecycle: both virtualizers, PeerScrollFollow,
 * UnitHighlight, multi-list FLIP layout, paired shift (`runPairedShift`),
 * scrollport chrome (end pads + card fill max + YAML header→scrollMargin), and In-card still-ness
 * (overflow lock + YAML→cards mute via phase machine).
 * View binds DOM roots, paints pad heights from session getters, and forwards UI
 * events; raw peerScroll / unitHighlight stay private.
 */
export function createTwinPaneSession(options: TwinPaneSessionOptions): TwinPaneSession {
	const createVirt = options.createComposerVirtualizer ?? createComposerVirtualizer;
	const createPeer = options.createPeerScrollFollow ?? ((opts) => new PeerScrollFollow(opts));
	const createHighlight = options.createUnitHighlight ?? ((inputs) => new UnitHighlight(inputs));
	const createLayout = options.createMultiListLayout ?? createMultiListLayout;
	const runPairedShift = options.runPairedShift ?? defaultRunPairedShift;
	const restoreScrollTop = options.restoreScrollTop ?? defaultRestoreScrollTop;
	const tick = options.tick ?? svelteTick;

	let cardsScroller = $state.raw<HTMLElement | null>(null);
	let yamlScroller = $state.raw<HTMLElement | null>(null);
	let stepsLayoutRoot = $state.raw<HTMLElement | null>(null);
	let yamlStepsLayoutRoot = $state.raw<HTMLElement | null>(null);
	/** Observer ~30% viewport pad; getter may raise it while In-card. */
	let cardsEndPadPx = $state(0);
	let cardsViewportPx = $state(0);
	let yamlEndPadPx = $state(0);
	/** Header height inside the YAML scroller — TanStack scrollMargin for step blocks. */
	let yamlScrollMargin = $state(0);
	let layout: CardListLayout | null = null;
	let lastFocusedCardToken = 0;
	let disposed = false;
	const inCard = createInCardSession();
	/** Public In-card surface — phase observation + sole unlock. */
	const inCardPublic: Pick<InCardSession, 'phase' | 'noteExitComplete'> = {
		get phase() {
			return inCard.phase;
		},
		noteExitComplete: () => inCard.noteExitComplete()
	};

	/** Derive In-card edit from editing index (set only for form + edit intent). */
	const getIsInCardEdit = () => options.getEditingIndex() !== undefined;
	const cardsEndPadAttach = endPadAttach(
		(px) => {
			cardsEndPadPx = px;
		},
		(viewportPx) => {
			cardsViewportPx = viewportPx;
		}
	);
	const yamlEndPadAttach = endPadAttach((px) => {
		yamlEndPadPx = px;
	});

	const listLengths = () => ({
		steps: options.getStepsCount(),
		followUps: options.getFollowUpsLength()
	});

	const stepsVirtualizer = createVirt({
		getCount: () => options.getStepsCount(),
		getScrollElement: () => cardsScroller,
		itemSelector: stepCardSelector,
		estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
		getItemKey: options.getItemKey
	});

	const yamlVirtualizer = createVirt({
		getCount: () => options.getYamlStepsCount(),
		getScrollElement: () => yamlScroller,
		getScrollMargin: () => yamlScrollMargin,
		itemSelector: yamlStepBlockSelector,
		estimateSize: () => DEFAULT_YAML_STEP_ESTIMATE_SIZE,
		getItemKey: options.getItemKey
	});

	const peerScroll = createPeer({
		ensureMounted: ensureMountedForStepsVirtualizer((index) =>
			stepsVirtualizer.ensureStepVisible(index, { behavior: 'auto' })
		),
		ensureMountedYaml: ensureMountedForStepsVirtualizer((index) =>
			yamlVirtualizer.ensureStepVisible(index, { behavior: 'auto' })
		),
		getCardLengths: listLengths,
		getYamlLengths: listLengths,
		isCardsParked: () => inCard.still
	});

	const unitHighlight = createHighlight({
		getIsManual: () => options.getIsManual(),
		getEditingIndex: () => options.getEditingIndex(),
		getEditingSection: () => options.getEditingSection()
	});

	options.bindComposerScroll({
		onRevealStep: (index) => {
			if (options.getIsManual()) return;
			const unit: ActiveUnit = { section: 'steps', index };
			void tick().then(() => peerScroll.onReveal(unit));
		},
		onEditFocus: (unit) => {
			if (options.getIsManual()) return;
			// Pencil chrome stops card onclick; pin here so YAML + card wash
			// stick after Save/dismiss (editingIndex clears, pin remains).
			// Do not followUnit — onEditFocus already start-aligns.
			unitHighlight.pinUnit(unit);
			inCard.noteEnterStart();
			void tick().then(async () => {
				if (disposed) {
					inCard.noteEnterSettled(false);
					return;
				}
				await peerScroll.onEditFocus(unit);
				if (disposed) {
					inCard.noteEnterSettled(false);
					return;
				}
				inCard.noteEnterSettled(getIsInCardEdit());
			});
		}
	});

	const measureStepCard: Attachment = (node) => {
		stepsVirtualizer.measureElement(node);
	};

	const measureYamlStep: Attachment = (node) => {
		yamlVirtualizer.measureElement(node);
	};

	const cardsParkWheelAttach: Attachment = (el) => {
		const onWheel = (event: WheelEvent) => {
			if (!inCard.still) return;
			if (nestedScrollerConsumesWheel(event, el)) return;
			event.preventDefault();
		};
		el.addEventListener('wheel', onWheel, { passive: false });
		return () => el.removeEventListener('wheel', onWheel);
	};

	/** Compose once — getters must not read mode or they invalidate `{@attach}`. */
	const cardsScrollAttach = composeAttachments(
		peerScroll.cardsAttach,
		cardsParkWheelAttach,
		cardsEndPadAttach
	);
	const yamlScrollAttach = composeAttachments(peerScroll.yamlAttach, yamlEndPadAttach);

	function setYamlHeaderHeight(height: number) {
		yamlScrollMargin = height;
	}

	function isCardSelected(section: ActiveUnit['section'], index: number): boolean {
		return unitHighlight.isCardSelected(section, index);
	}

	function isCardHovered(section: ActiveUnit['section'], index: number): boolean {
		return unitHighlight.isCardHovered(section, index);
	}

	function hoverCard(unit: ActiveUnit): void {
		unitHighlight.hoverCard(unit);
	}

	function clearHoverCard(unit: ActiveUnit): void {
		unitHighlight.clearHoverCard(unit);
	}

	/**
	 * Named swap-mount: sole owner of mountOnly + align:start + dual-pane
	 * scrollTop/scrollOffset pin for paired shift. Paired-shift only sees the
	 * opaque `ensureSwapMounted` callback — it must not pass mountOnly or virt opts.
	 *
	 * Uses `mountOnly` so already-in-DOM units do not `scrollToIndex` (align:start +
	 * TanStack scroll reconcile would yank mid-list / top-of-list pins toward 0).
	 * When a scroll-to-mount did run, snapshot+restore still pins scrollTop before
	 * `runPairedShift`.
	 */
	async function mountSwapIndices(fromIndex: number, toIndex: number) {
		const cardsTop = cardsScroller?.scrollTop ?? 0;
		const yamlTop = yamlScroller?.scrollTop ?? 0;
		const mountOpts = {
			behavior: 'auto' as const,
			align: 'start' as const,
			mountOnly: true
		};
		await Promise.all([
			stepsVirtualizer.ensureStepVisible(fromIndex, mountOpts),
			stepsVirtualizer.ensureStepVisible(toIndex, mountOpts),
			yamlVirtualizer.ensureStepVisible(fromIndex, mountOpts),
			yamlVirtualizer.ensureStepVisible(toIndex, mountOpts)
		]);
		pinBothAt(cardsTop, yamlTop);
		await tick();
	}

	/** DOM+TanStack pin for both panes — shared by mount-swap and paired-shift FLIP. */
	function pinBothAt(cardsTop: number, yamlTop: number) {
		pinBothScrollports(
			{
				scroller: cardsScroller,
				top: cardsTop,
				virtualizer: get(stepsVirtualizer.virtualizer)
			},
			{
				scroller: yamlScroller,
				top: yamlTop,
				virtualizer: get(yamlVirtualizer.virtualizer)
			},
			restoreScrollTop
		);
	}

	/** Rememo both cards+yaml virtualizers (`itemKeyFn` identity bump) after a paired mutate. */
	function rememoAfterPairedMutate() {
		stepsVirtualizer.syncAfterReorder();
		yamlVirtualizer.syncAfterReorder();
	}

	async function shiftStep(index: number, change: number) {
		if (disposed) return;
		if (!options.canShiftStep(index, change)) return;
		const toIndex = index + change;
		await runPairedShift({
			fromIndex: index,
			toIndex,
			notePairedReorder: () => peerScroll.notePairedReorder(),
			ensureSwapMounted: mountSwapIndices,
			cardsScroller,
			yamlScroller,
			layout,
			mutate: () => {
				options.mutateShiftStep(index, change);
				unitHighlight.remapStepsAfterAdjacentSwap(index, toIndex);
			},
			rememoAfterMutate: rememoAfterPairedMutate,
			pinScrollports: pinBothAt,
			tick
		});
	}

	function onUnitClick(unit: ActiveUnit, from: 'cards' | 'yaml') {
		if (disposed) return;
		const pinned = unitHighlight.pinUnit(unit);
		if (!pinned) return;
		peerScroll.followUnit(pinned, from);
	}

	function onYamlUnitHover(unit: ActiveUnit | null) {
		if (disposed) return;
		if (unit === null) {
			unitHighlight.clearHover();
			return;
		}
		unitHighlight.hoverCard(unit);
	}

	function setFollowEnabled(checked: boolean) {
		if (disposed) return;
		peerScroll.setEnabled(
			checked,
			options.getEditingSection() === 'steps' ? options.getEditingIndex() : undefined
		);
	}

	const destroyEffects = $effect.root(() => {
		$effect(() => {
			const cardsRoot = stepsLayoutRoot;
			if (!cardsRoot) return;
			const yamlRoot = yamlStepsLayoutRoot;
			const created = createLayout([
				{ root: cardsRoot },
				...(yamlRoot ? [{ root: yamlRoot, children: DEFAULT_YAML_LAYOUT_CHILDREN }] : [])
			]);
			layout = created;
			return () => {
				created.dispose();
				if (layout === created) layout = null;
			};
		});

		$effect(() => {
			const createdCard = options.getCreatedCard();
			if (!createdCard || createdCard.token === lastFocusedCardToken) return;

			lastFocusedCardToken = createdCard.token;
			void tick().then(async () => {
				if (disposed) return;
				if (createdCard.section === 'steps') {
					await stepsVirtualizer.ensureStepVisible(createdCard.index, {
						align: 'auto',
						behavior: 'smooth'
					});
				}
				const scroller = cardsScroller;
				const selector = `[data-card-section="${createdCard.section}"][data-card-index="${createdCard.index}"]`;
				const card =
					scroller?.querySelector<HTMLElement>(selector) ??
					document.querySelector<HTMLElement>(selector);
				if (createdCard.section !== 'steps') {
					card?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
				}
				card?.focus({ preventScroll: true });
			});
		});

		$effect(() => {
			if (!peerScroll.enabled || options.getIsManual()) return;
			const yaml = options.getYamlPreview();
			const yamlSteps = options.getYamlStepsCount();
			if (!yaml || yamlSteps === 0) return;
			return peerScroll.onYamlTextChanged();
		});

		/** Form mode left while still → exiting; unlock waits for noteExitComplete. */
		$effect(() => {
			const phase = inCard.phase;
			const editing = options.getEditingIndex();
			if (phase !== 'still') return;
			if (editing !== undefined) return;
			inCard.noteEditingEnded();
		});
	});

	function dispose() {
		if (disposed) return;
		disposed = true;
		inCard.dispose();
		destroyEffects();
		options.bindComposerScroll({});
		peerScroll.dispose();
		unitHighlight.dispose();
		stepsVirtualizer.dispose();
		yamlVirtualizer.dispose();
		layout?.dispose();
		layout = null;
	}

	return {
		get cardsScroller() {
			return cardsScroller;
		},
		set cardsScroller(el) {
			cardsScroller = el;
		},
		get yamlScroller() {
			return yamlScroller;
		},
		set yamlScroller(el) {
			yamlScroller = el;
		},
		get stepsLayoutRoot() {
			return stepsLayoutRoot;
		},
		set stepsLayoutRoot(el) {
			stepsLayoutRoot = el;
		},
		get yamlStepsLayoutRoot() {
			return yamlStepsLayoutRoot;
		},
		set yamlStepsLayoutRoot(el) {
			yamlStepsLayoutRoot = el;
		},
		get cardsEndPadPx() {
			return inCardCardsEndPadPx(cardsEndPadPx, cardsViewportPx, inCard.phase !== 'idle');
		},
		get cardsViewportPx() {
			return cardsViewportPx;
		},
		get cardFillMaxPx() {
			return inCardCardFillMaxPx(cardsViewportPx);
		},
		get yamlEndPadPx() {
			return yamlEndPadPx;
		},
		get yamlScrollMargin() {
			return yamlScrollMargin;
		},
		stepsVirt: stepsVirtualizer.virtualizer,
		yamlVirt: yamlVirtualizer.virtualizer,
		measureStepCard,
		measureYamlStep,
		get cardsScrollAttach() {
			return cardsScrollAttach;
		},
		get yamlScrollAttach() {
			return yamlScrollAttach;
		},
		setYamlHeaderHeight,
		isCardSelected,
		isCardHovered,
		hoverCard,
		clearHoverCard,
		get followEnabled() {
			return peerScroll.enabled;
		},
		inCard: inCardPublic,
		get still() {
			return inCard.still;
		},
		shiftStep,
		onUnitClick,
		onYamlUnitHover,
		setFollowEnabled,
		dispose
	};
}
