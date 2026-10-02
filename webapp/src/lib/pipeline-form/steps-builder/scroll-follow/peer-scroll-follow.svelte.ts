// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Attachment } from 'svelte/attachments';

import {
	resolveViewportActiveCard,
	resolveViewportYamlLine,
	sameUnit,
	scrollCardIntoView,
	scrollYamlLineIntoView,
	watchDrivenScroll,
	type ActiveUnit,
	type ScrollAlign
} from './active-unit.js';
import { scrollFollowPreference } from './preference.js';
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

/**
 * Viewport peer sync for Pipeline Composer cards ↔ YAML.
 * Does not select or highlight a step (see UnitHighlight).
 */
export class PeerScrollFollow {
	activeUnit = $state.raw<ActiveUnit | null>(null);

	#clock: PeerScrollFollowClock;
	#cardsEl: HTMLElement | null = null;
	#yamlEl: HTMLElement | null = null;
	#getRanges: (() => YamlCardRange[]) | null = null;

	/** Per-side programmatic-scroll guards; hard start drives both sides at once. */
	#clearDriven: Record<'cards' | 'yaml', (() => void) | null> = { cards: null, yaml: null };
	#scrollLeader: 'cards' | 'yaml' | null = null;
	#lastIntentSide: 'cards' | 'yaml' | null = null;
	#leaderIdleTimer: ReturnType<typeof setTimeout> | null = null;
	#peerFollowRaf: number | null = null;
	#pendingReveal: ActiveUnit | null = null;
	#disposed = false;

	constructor(options?: PeerScrollFollowOptions) {
		this.#clock = options?.clock ?? defaultClock();
	}

	/** Shared persisted preference (rune-sync / localStorage). */
	get enabled(): boolean {
		return scrollFollowPreference.enabled;
	}

	setEnabled(next: boolean, editingUnit?: ActiveUnit | null) {
		if (this.#disposed) return;
		scrollFollowPreference.enabled = next;
		if (!next) {
			this.activeUnit = null;
			return;
		}
		if (editingUnit) {
			this.#setActiveUnit(editingUnit);
			this.#followPeerFromCards('smooth');
		}
	}

	/** Attachment for the cards scrollport (Column). Safe while follow is off — handlers no-op. */
	cardsAttach: Attachment = (node) => {
		const el = node as HTMLElement;
		this.#cardsEl = el;

		const onUserIntent = () => {
			if (!this.enabled) return;
			this.#clearDriven.cards?.();
			this.#lastIntentSide = 'cards';
			this.#claimScrollLeader('cards');
		};

		const onScroll = () => {
			if (!this.enabled) return;
			if (this.#clearDriven.cards) return;
			if (this.#scrollLeader === 'yaml') return;
			if (this.#scrollLeader !== 'cards' && this.#lastIntentSide !== 'cards') return;
			this.#claimScrollLeader('cards');
			const next = resolveViewportActiveCard(el, this.activeUnit);
			if (!next || sameUnit(this.activeUnit, next)) return;
			this.#setActiveUnit(next);
			this.#schedulePeerFollow('cards');
		};

		el.addEventListener('wheel', onUserIntent, { passive: true });
		el.addEventListener('touchstart', onUserIntent, { passive: true });
		el.addEventListener('pointerdown', onUserIntent, { passive: true });
		el.addEventListener('scroll', onScroll, { passive: true });

		const pending = this.#pendingReveal;
		if (pending) {
			this.#pendingReveal = null;
			this.#flushReveal(pending);
		}

		return () => {
			el.removeEventListener('wheel', onUserIntent);
			el.removeEventListener('touchstart', onUserIntent);
			el.removeEventListener('pointerdown', onUserIntent);
			el.removeEventListener('scroll', onScroll);
			if (this.#cardsEl === el) this.#cardsEl = null;
		};
	};

	/**
	 * Attachment factory for the YAML scroller (CodeDisplay).
	 * `getRanges` is read on scroll — do not recreate the attachment on every YAML regen.
	 */
	yamlAttach(getRanges: () => YamlCardRange[]): Attachment {
		return (node) => {
			const el = node as HTMLElement;
			this.#yamlEl = el;
			this.#getRanges = getRanges;

			const onUserIntent = () => {
				if (!this.enabled) return;
				this.#clearDriven.yaml?.();
				this.#lastIntentSide = 'yaml';
				this.#claimScrollLeader('yaml');
			};

			const onScroll = () => {
				if (!this.enabled) return;
				if (this.#clearDriven.yaml) return;
				if (this.#scrollLeader === 'cards') return;
				if (this.#scrollLeader !== 'yaml' && this.#lastIntentSide !== 'yaml') return;
				this.#claimScrollLeader('yaml');
				const ranges = this.#getRanges?.() ?? [];
				const firstStep = firstStepStartLine(ranges);
				const line = resolveViewportYamlLine(el, firstStep);
				if (line === null) {
					this.#setActiveUnit(null);
					return;
				}
				const hit = findNearestUnitToLine(ranges, line);
				if (!hit) {
					this.#setActiveUnit(null);
					return;
				}
				const next: ActiveUnit = { section: hit.section, index: hit.index };
				if (sameUnit(this.activeUnit, next)) return;
				this.#setActiveUnit(next);
				this.#schedulePeerFollow('yaml');
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
				if (this.#yamlEl === el) {
					this.#yamlEl = null;
					this.#getRanges = null;
				}
			};
		};
	}

	/**
	 * Enter hard start for In-card edit: start-align the card and its YAML range, regardless
	 * of the Scroll follow preference. One-shot — does not pin or suspend peer sync afterwards.
	 * Resolves when the cards pane scroll has settled (or immediately if no scroll was needed),
	 * so callers can sequence expand-after-scroll.
	 */
	onEditFocus(unit: ActiveUnit): Promise<void> {
		if (this.#disposed) return Promise.resolve();
		this.#setActiveUnit(unit);
		const cards = this.#cardsEl;
		let cardsSettled = Promise.resolve();
		if (cards) {
			cardsSettled = new Promise<void>((resolve) => {
				this.#clearDriven.cards?.();
				const scrolled = scrollCardIntoView(cards, unit, 'smooth', {
					align: 'start',
					focus: false
				});
				if (!scrolled) {
					resolve();
					return;
				}
				// Drive-guard + settle signal (same timeout as #beginDriven / watchDrivenScroll).
				this.#claimScrollLeader('yaml');
				this.#clearDriven.cards = watchDrivenScroll(
					cards,
					'smooth',
					() => {
						this.#clearDriven.cards = null;
						this.#claimScrollLeader('yaml');
						resolve();
					},
					this.#clock
				);
			});
		}
		this.#scrollYamlToUnit(unit, 'smooth', 'start');
		return cardsSettled;
	}

	/** Center-scroll card; if enabled also peer-follow YAML. Queues if cards not bound yet. */
	onReveal(unit: ActiveUnit) {
		if (this.#disposed) return;
		if (!this.#cardsEl) {
			this.#pendingReveal = unit;
			return;
		}
		this.#flushReveal(unit);
	}

	#flushReveal(unit: ActiveUnit) {
		const cards = this.#cardsEl;
		if (!cards) return;
		scrollCardIntoView(cards, unit, 'smooth', { align: 'center', focus: false });
		if (!this.enabled) return;
		this.#setActiveUnit(unit);
		this.#followPeerFromCards('smooth');
	}

	/** Debounced re-follow from cards; returns cancel cleanup. */
	onYamlTextChanged(): () => void {
		if (this.#disposed || !this.enabled) return () => {};
		const timer = this.#clock.setTimeout(() => {
			if (!this.activeUnit) return;
			this.#followPeerFromCards('smooth');
		}, YAML_REGEN_DEBOUNCE_MS);
		return () => this.#clock.clearTimeout(timer);
	}

	/** After view pins selection from YAML click. */
	followUnit(unit: ActiveUnit, from: 'cards' | 'yaml') {
		if (this.#disposed) return;
		this.#setActiveUnit(unit);
		if (!this.enabled) return;
		if (from === 'yaml') {
			this.#lastIntentSide = 'yaml';
			this.#claimScrollLeader('yaml');
			this.#followPeerFromYaml('smooth');
			return;
		}
		this.#lastIntentSide = 'cards';
		this.#claimScrollLeader('cards');
		this.#followPeerFromCards('smooth');
	}

	dispose() {
		if (this.#disposed) return;
		this.#disposed = true;
		this.#clearDriven.cards?.();
		this.#clearDriven.yaml?.();
		if (this.#leaderIdleTimer) {
			this.#clock.clearTimeout(this.#leaderIdleTimer);
			this.#leaderIdleTimer = null;
		}
		if (this.#peerFollowRaf != null) {
			this.#clock.cancelRaf(this.#peerFollowRaf);
			this.#peerFollowRaf = null;
		}
		this.#scrollLeader = null;
		this.#lastIntentSide = null;
		this.#pendingReveal = null;
		this.#cardsEl = null;
		this.#yamlEl = null;
		this.#getRanges = null;
	}

	#setActiveUnit(next: ActiveUnit | null) {
		if (sameUnit(this.activeUnit, next)) return;
		this.activeUnit = next;
	}

	#claimScrollLeader(side: 'cards' | 'yaml') {
		this.#scrollLeader = side;
		if (this.#leaderIdleTimer) this.#clock.clearTimeout(this.#leaderIdleTimer);
		this.#leaderIdleTimer = this.#clock.setTimeout(() => {
			this.#scrollLeader = null;
			this.#leaderIdleTimer = null;
		}, LEADER_IDLE_MS);
	}

	#beginDriven(side: 'cards' | 'yaml', el: HTMLElement, behavior: ScrollBehavior) {
		this.#clearDriven[side]?.();
		const leader: 'cards' | 'yaml' = side === 'cards' ? 'yaml' : 'cards';
		this.#claimScrollLeader(leader);
		this.#clearDriven[side] = watchDrivenScroll(
			el,
			behavior,
			() => {
				this.#clearDriven[side] = null;
				this.#claimScrollLeader(leader);
			},
			this.#clock
		);
	}

	#followPeerFromCards(behavior: ScrollBehavior) {
		if (this.#scrollLeader === 'yaml') return;
		const unit = this.activeUnit;
		if (!unit) return;
		this.#scrollYamlToUnit(
			unit,
			behavior,
			this.#scrollLeader === 'cards' ? 'start-band' : 'start'
		);
	}

	#scrollYamlToUnit(unit: ActiveUnit, behavior: ScrollBehavior, align: ScrollAlign) {
		const yaml = this.#yamlEl;
		if (!yaml) return;
		const ranges = this.#getRanges?.() ?? [];
		const range = findRangeForUnit(ranges, unit.section, unit.index);
		if (!range) return;
		this.#beginDriven('yaml', yaml, behavior);
		if (!scrollYamlLineIntoView(yaml, range.startLine, behavior, align)) {
			this.#clearDriven.yaml?.();
		}
	}

	#followPeerFromYaml(behavior: ScrollBehavior) {
		const unit = this.activeUnit;
		const cards = this.#cardsEl;
		if (!unit || !cards) return;
		this.#beginDriven('cards', cards, behavior);
		const align = 'center';
		if (
			!scrollCardIntoView(cards, unit, behavior, {
				align,
				focus: behavior === 'smooth'
			})
		) {
			this.#clearDriven.cards?.();
		}
	}

	#schedulePeerFollow(from: 'cards' | 'yaml') {
		if (this.#peerFollowRaf != null) this.#clock.cancelRaf(this.#peerFollowRaf);
		this.#peerFollowRaf = this.#clock.raf(() => {
			this.#peerFollowRaf = null;
			if (from === 'cards') this.#followPeerFromCards('smooth');
			else this.#followPeerFromYaml('smooth');
		});
	}
}
