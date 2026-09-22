// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	resolveViewportActiveCard,
	resolveViewportYamlLine,
	sameUnit,
	scrollCardIntoView,
	scrollYamlLineIntoView,
	watchDrivenScroll,
	type ActiveUnit
} from './active-unit.js';
import { readScrollFollowEnabled, writeScrollFollowEnabled } from './preference.js';
import {
	findNearestUnitToLine,
	findRangeForUnit,
	firstStepStartLine,
	type YamlCardRange
} from './yaml-ranges.js';

export type PeerScrollFollowClock = {
	raf: (callback: FrameRequestCallback) => number;
	cancelRaf: (handle: number) => void;
	setTimeout: (handler: () => void, timeout?: number) => ReturnType<typeof setTimeout>;
	clearTimeout: (handle: ReturnType<typeof setTimeout>) => void;
};

export type PeerScrollFollowOptions = {
	clock?: PeerScrollFollowClock;
};

export type PeerScrollFollow = {
	readonly enabled: boolean;
	readonly activeUnit: ActiveUnit | null;
	subscribe(listener: () => void): () => void;
	setEnabled(enabled: boolean, editingIndex?: number): void;
	bindCards(el: HTMLElement): () => void;
	bindYaml(el: HTMLElement, getRanges: () => YamlCardRange[]): () => void;
	onEditFocus(stepIndex: number | undefined): void;
	/** Center-scroll card; if enabled also peer-follow YAML. */
	onReveal(unit: ActiveUnit): void;
	/** Debounced re-follow from cards; returns cancel cleanup. */
	onYamlTextChanged(): () => void;
	/** After view pins selection from YAML click. */
	followUnit(unit: ActiveUnit, from: 'cards' | 'yaml'): void;
	dispose(): void;
};

const YAML_REGEN_DEBOUNCE_MS = 130;
const LEADER_IDLE_MS = 900;

function defaultClock(): PeerScrollFollowClock {
	return {
		raf: (...args) => globalThis.requestAnimationFrame(...args),
		cancelRaf: (...args) => globalThis.cancelAnimationFrame(...args),
		setTimeout: (...args) => globalThis.setTimeout(...args),
		clearTimeout: (...args) => globalThis.clearTimeout(...args)
	};
}

export function createPeerScrollFollow(options?: PeerScrollFollowOptions): PeerScrollFollow {
	const clock = options?.clock ?? defaultClock();

	let enabled = readScrollFollowEnabled();
	let activeUnit: ActiveUnit | null = null;
	const listeners = new Set<() => void>();

	let cardsEl: HTMLElement | null = null;
	let yamlEl: HTMLElement | null = null;
	let getRanges: (() => YamlCardRange[]) | null = null;

	let drivenSide: 'cards' | 'yaml' | null = null;
	let clearDriven: (() => void) | null = null;
	let scrollLeader: 'cards' | 'yaml' | null = null;
	let lastIntentSide: 'cards' | 'yaml' | null = null;
	let leaderIdleTimer: ReturnType<typeof setTimeout> | null = null;
	let peerFollowRaf: number | null = null;

	let disposed = false;

	function notify() {
		for (const listener of listeners) listener();
	}

	function setActiveUnit(next: ActiveUnit | null) {
		if (sameUnit(activeUnit, next)) return;
		activeUnit = next;
		notify();
	}

	function claimScrollLeader(side: 'cards' | 'yaml') {
		scrollLeader = side;
		if (leaderIdleTimer) clock.clearTimeout(leaderIdleTimer);
		leaderIdleTimer = clock.setTimeout(() => {
			scrollLeader = null;
			leaderIdleTimer = null;
		}, LEADER_IDLE_MS);
	}

	function beginDriven(side: 'cards' | 'yaml', el: HTMLElement, behavior: ScrollBehavior) {
		clearDriven?.();
		drivenSide = side;
		const leader: 'cards' | 'yaml' = side === 'cards' ? 'yaml' : 'cards';
		claimScrollLeader(leader);
		clearDriven = watchDrivenScroll(
			el,
			behavior,
			() => {
				if (drivenSide === side) drivenSide = null;
				clearDriven = null;
				claimScrollLeader(leader);
			},
			clock
		);
	}

	function followPeerFromCards(behavior: ScrollBehavior) {
		if (scrollLeader === 'yaml') return;
		const unit = activeUnit;
		const yaml = yamlEl;
		if (!unit || !yaml) return;
		const ranges = getRanges?.() ?? [];
		const range = findRangeForUnit(ranges, unit.section, unit.index);
		if (!range) return;
		beginDriven('yaml', yaml, behavior);
		const align = scrollLeader === 'cards' ? 'start-band' : 'start';
		if (!scrollYamlLineIntoView(yaml, range.startLine, behavior, align)) {
			clearDriven?.();
		}
	}

	function followPeerFromYaml(behavior: ScrollBehavior) {
		const unit = activeUnit;
		const cards = cardsEl;
		if (!unit || !cards) return;
		beginDriven('cards', cards, behavior);
		const align = 'center';
		if (
			!scrollCardIntoView(cards, unit, behavior, {
				align,
				focus: behavior === 'smooth'
			})
		) {
			clearDriven?.();
		}
	}

	function schedulePeerFollow(from: 'cards' | 'yaml') {
		if (peerFollowRaf != null) clock.cancelRaf(peerFollowRaf);
		peerFollowRaf = clock.raf(() => {
			peerFollowRaf = null;
			if (from === 'cards') followPeerFromCards('smooth');
			else followPeerFromYaml('smooth');
		});
	}

	function subscribe(listener: () => void): () => void {
		listeners.add(listener);
		return () => {
			listeners.delete(listener);
		};
	}

	function setEnabled(next: boolean, editingIndex?: number) {
		if (disposed) return;
		enabled = next;
		writeScrollFollowEnabled(next);
		if (!next) {
			activeUnit = null;
			notify();
			return;
		}
		notify();
		if (editingIndex !== undefined) {
			setActiveUnit({ section: 'steps', index: editingIndex });
			followPeerFromCards('smooth');
		}
	}

	function bindCards(el: HTMLElement): () => void {
		cardsEl = el;

		const onUserIntent = () => {
			if (!enabled) return;
			if (drivenSide === 'cards') clearDriven?.();
			lastIntentSide = 'cards';
			claimScrollLeader('cards');
		};

		const onScroll = () => {
			if (!enabled) return;
			if (drivenSide === 'cards') return;
			if (scrollLeader === 'yaml') return;
			if (scrollLeader !== 'cards' && lastIntentSide !== 'cards') return;
			claimScrollLeader('cards');
			const next = resolveViewportActiveCard(el, activeUnit);
			if (!next || sameUnit(activeUnit, next)) return;
			setActiveUnit(next);
			schedulePeerFollow('cards');
		};

		el.addEventListener('wheel', onUserIntent, { passive: true });
		el.addEventListener('touchstart', onUserIntent, { passive: true });
		el.addEventListener('pointerdown', onUserIntent, { passive: true });
		el.addEventListener('scroll', onScroll, { passive: true });

		return () => {
			el.removeEventListener('wheel', onUserIntent);
			el.removeEventListener('touchstart', onUserIntent);
			el.removeEventListener('pointerdown', onUserIntent);
			el.removeEventListener('scroll', onScroll);
			if (cardsEl === el) cardsEl = null;
		};
	}

	function bindYaml(el: HTMLElement, rangesFn: () => YamlCardRange[]): () => void {
		yamlEl = el;
		getRanges = rangesFn;

		const onUserIntent = () => {
			if (!enabled) return;
			if (drivenSide === 'yaml') clearDriven?.();
			lastIntentSide = 'yaml';
			claimScrollLeader('yaml');
		};

		const onScroll = () => {
			if (!enabled) return;
			if (drivenSide === 'yaml') return;
			if (scrollLeader === 'cards') return;
			if (scrollLeader !== 'yaml' && lastIntentSide !== 'yaml') return;
			claimScrollLeader('yaml');
			const ranges = getRanges?.() ?? [];
			const firstStep = firstStepStartLine(ranges);
			const line = resolveViewportYamlLine(el, firstStep);
			if (line === null) {
				setActiveUnit(null);
				return;
			}
			const hit = findNearestUnitToLine(ranges, line);
			if (!hit) {
				setActiveUnit(null);
				return;
			}
			const next: ActiveUnit = { section: hit.section, index: hit.index };
			if (sameUnit(activeUnit, next)) return;
			setActiveUnit(next);
			schedulePeerFollow('yaml');
		};

		el.addEventListener('wheel', onUserIntent, { passive: true });
		el.addEventListener('touchstart', onUserIntent, { passive: true });
		el.addEventListener('pointerdown', onUserIntent, { passive: true });
		el.addEventListener('scroll', onScroll, { passive: true });

		return () => {
			el.removeEventListener('wheel', onUserIntent);
			el.removeEventListener('touchstart', onUserIntent);
			el.removeEventListener('pointerdown', onUserIntent);
			el.removeEventListener('scroll', onScroll);
			if (yamlEl === el) {
				yamlEl = null;
				getRanges = null;
			}
		};
	}

	function onEditFocus(stepIndex: number | undefined) {
		if (disposed) return;
		if (stepIndex === undefined) return;
		const next: ActiveUnit = { section: 'steps', index: stepIndex };
		if (sameUnit(activeUnit, next)) return;
		setActiveUnit(next);
		if (enabled) followPeerFromCards('smooth');
	}

	function onReveal(unit: ActiveUnit) {
		if (disposed) return;
		const cards = cardsEl;
		if (!cards) return;
		scrollCardIntoView(cards, unit, 'smooth', { align: 'center', focus: false });
		if (!enabled) return;
		setActiveUnit(unit);
		followPeerFromCards('smooth');
	}

	function onYamlTextChanged(): () => void {
		if (disposed || !enabled) return () => {};
		const timer = clock.setTimeout(() => {
			if (!activeUnit) return;
			followPeerFromCards('smooth');
		}, YAML_REGEN_DEBOUNCE_MS);
		return () => clock.clearTimeout(timer);
	}

	function followUnit(unit: ActiveUnit, from: 'cards' | 'yaml') {
		if (disposed) return;
		setActiveUnit(unit);
		if (!enabled) return;
		if (from === 'yaml') {
			lastIntentSide = 'yaml';
			claimScrollLeader('yaml');
			followPeerFromYaml('smooth');
			return;
		}
		lastIntentSide = 'cards';
		claimScrollLeader('cards');
		followPeerFromCards('smooth');
	}

	function dispose() {
		if (disposed) return;
		disposed = true;
		clearDriven?.();
		clearDriven = null;
		drivenSide = null;
		if (leaderIdleTimer) {
			clock.clearTimeout(leaderIdleTimer);
			leaderIdleTimer = null;
		}
		if (peerFollowRaf != null) {
			clock.cancelRaf(peerFollowRaf);
			peerFollowRaf = null;
		}
		scrollLeader = null;
		lastIntentSide = null;
		cardsEl = null;
		yamlEl = null;
		getRanges = null;
		listeners.clear();
	}

	return {
		get enabled() {
			return enabled;
		},
		get activeUnit() {
			return activeUnit;
		},
		subscribe,
		setEnabled,
		bindCards,
		bindYaml,
		onEditFocus,
		onReveal,
		onYamlTextChanged,
		followUnit,
		dispose
	};
}
