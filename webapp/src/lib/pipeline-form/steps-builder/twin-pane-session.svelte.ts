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
	restoreScrollTop as defaultRestoreScrollTop,
	stepCardSelector,
	yamlStepBlockSelector,
	type ComposerVirtualizer,
	type ComposerVirtualizerOptions
} from './composer-virtualizer.svelte.js';
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
	type ActiveUnit
} from './scroll-follow/active-unit.js';
import {
	PeerScrollFollow,
	type PeerScrollFollowOptions
} from './scroll-follow/peer-scroll-follow.svelte.js';
import {
	UnitHighlight,
	type UnitHighlightInputs
} from './scroll-follow/unit-highlight.svelte.js';

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
	/** TanStack scrollMargin for YAML step blocks (header height). */
	getYamlScrollMargin?: () => number;
	canShiftStep: (index: number, change: number) => boolean;
	mutateShiftStep: (index: number, change: number) => void;
	bindComposerScroll: (handlers: {
		onRevealStep?: (index: number) => void;
		onEditFocus?: (stepIndex: number) => void;
	}) => void;
	/** Inject for tests. */
	createComposerVirtualizer?: (
		options: ComposerVirtualizerOptions
	) => ComposerVirtualizer;
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
	/** TanStack Readable — `$stepsVirt` in markup. */
	stepsVirt: ComposerVirtualizer['virtualizer'];
	stepsVirtualizer: ComposerVirtualizer;
	yamlVirtualizer: ComposerVirtualizer;
	measureStepCard: Attachment;
	peerScroll: PeerScrollFollow;
	unitHighlight: UnitHighlight;
	shiftStep(index: number, change: number): Promise<void>;
	onUnitClick(unit: ActiveUnit, from: 'cards' | 'yaml'): void;
	onYamlUnitHover(unit: ActiveUnit | null): void;
	setFollowEnabled(checked: boolean): void;
	dispose(): void;
};

/**
 * Owns Pipeline Composer twin-pane lifecycle: both virtualizers, PeerScrollFollow,
 * UnitHighlight, multi-list FLIP layout, and paired shift (`runPairedShift`).
 * View binds DOM roots and forwards UI events only.
 */
export function createTwinPaneSession(options: TwinPaneSessionOptions): TwinPaneSession {
	const createVirt = options.createComposerVirtualizer ?? createComposerVirtualizer;
	const createPeer = options.createPeerScrollFollow ?? ((opts) => new PeerScrollFollow(opts));
	const createHighlight = options.createUnitHighlight ?? ((inputs) => new UnitHighlight(inputs));
	const createLayout = options.createMultiListLayout ?? createMultiListLayout;
	const runPairedShift = options.runPairedShift ?? defaultRunPairedShift;
	const restoreScrollTop = options.restoreScrollTop ?? defaultRestoreScrollTop;
	const tick = options.tick ?? svelteTick;
	const getYamlScrollMargin = options.getYamlScrollMargin ?? (() => 0);

	let cardsScroller = $state.raw<HTMLElement | null>(null);
	let yamlScroller = $state.raw<HTMLElement | null>(null);
	let stepsLayoutRoot = $state.raw<HTMLElement | null>(null);
	let yamlStepsLayoutRoot = $state.raw<HTMLElement | null>(null);
	let layout: CardListLayout | null = null;
	let lastFocusedCardToken = 0;
	let disposed = false;

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
		getScrollMargin: () => getYamlScrollMargin(),
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
		getYamlLengths: listLengths
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
		onEditFocus: (stepIndex) => {
			if (options.getIsManual()) return;
			peerScroll.onEditFocus(stepIndex);
		}
	});

	const measureStepCard: Attachment = (node) => {
		stepsVirtualizer.measureElement(node);
	};

	/**
	 * Mount both swap indices on cards and YAML before FLIP.
	 * Uses `mountOnly` so already-in-DOM units do not `scrollToIndex` (align:start +
	 * TanStack scroll reconcile would yank mid-list / top-of-list pins toward 0).
	 * When a scroll-to-mount did run, snapshot+restore still pins scrollTop before
	 * `runPairedShift`.
	 */
	async function ensureSwapIndicesMounted(fromIndex: number, toIndex: number) {
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
		restoreScrollTop(cardsScroller, cardsTop, get(stepsVirtualizer.virtualizer));
		restoreScrollTop(yamlScroller, yamlTop, get(yamlVirtualizer.virtualizer));
		await tick();
	}

	async function shiftStep(index: number, change: number) {
		if (disposed) return;
		if (!options.canShiftStep(index, change)) return;
		const toIndex = index + change;
		await runPairedShift({
			fromIndex: index,
			toIndex,
			notePairedReorder: () => peerScroll.notePairedReorder(),
			ensureSwapMounted: ensureSwapIndicesMounted,
			cardsScroller,
			yamlScroller,
			layout,
			mutate: () => options.mutateShiftStep(index, change),
			syncBoth: () => {
				stepsVirtualizer.syncAfterReorder();
				yamlVirtualizer.syncAfterReorder();
			},
			tick,
			restoreScrollTop
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
	});

	function dispose() {
		if (disposed) return;
		disposed = true;
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
		stepsVirt: stepsVirtualizer.virtualizer,
		stepsVirtualizer,
		yamlVirtualizer,
		measureStepCard,
		peerScroll,
		unitHighlight,
		shiftStep,
		onUnitClick,
		onYamlUnitHover,
		setFollowEnabled,
		dispose
	};
}
