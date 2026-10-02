// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Attachment } from 'svelte/attachments';

import { browserClock, type ComposerClock } from '../composer-clock.js';
import {
	CARD_PANE,
	YAML_PANE,
	resolveTopmostVisibleUnit,
	resolveViewportUnit,
	sameUnit,
	scrollUnitIntoView,
	watchDrivenScroll,
	type ActiveUnit,
	type CardListLengths,
	type EnsureMounted
} from './active-unit.js';
import {
	animatableDrivenIdleMs,
	createAnimatableScroll,
	DISCRETE_SCROLL_DURATION_MS,
	FOLLOW_SCROLL_DURATION_MS,
	type AnimatableScroll
} from './animatable-scroll.js';
import { scrollFollowPreference } from './preference.js';

/** Continuous retarget vs intentional reveal/edit/click (duration only; both use Animatable). */
export type PeerScrollKind = 'follow' | 'discrete';

export type PeerScrollFollowOptions = {
	clock?: ComposerClock;
	/**
	 * Optional mount hook for unmounted (virtualized) cards.
	 * Behavior-compatible when absent — missing cards still no-op scroll.
	 */
	ensureMounted?: EnsureMounted;
	/**
	 * Optional mount hook for unmounted (virtualized) YAML step blocks.
	 * Follow-ups in the YAML pane stay fully mounted.
	 */
	ensureMountedYaml?: EnsureMounted;
	/** Optional true list lengths for list-end viewport resolution under virtualization. */
	getCardLengths?: () => CardListLengths | undefined;
	/** Optional true YAML list lengths (usually same as cards). */
	getYamlLengths?: () => CardListLengths | undefined;
	/** Injectable Animatable factory (tests). Defaults to animejs `createAnimatable`. */
	createAnimatableScroll?: typeof createAnimatableScroll;
};

/** Debounce before cards→YAML discrete follow after preview text regenerates. */
export const YAML_REGEN_DEBOUNCE_MS = 130;
/** Slack past debounce so paired-reorder regen does not schedule a competing follow. */
const PAIRED_REORDER_SUPPRESS_MS = YAML_REGEN_DEBOUNCE_MS + 50;
const LEADER_IDLE_MS = 900;

function durationFor(kind: PeerScrollKind): number {
	return kind === 'follow' ? FOLLOW_SCROLL_DURATION_MS : DISCRETE_SCROLL_DURATION_MS;
}

/**
 * Viewport peer sync for Pipeline Composer cards ↔ YAML (index-aligned).
 * Does not select or highlight a step (see UnitHighlight).
 *
 * Smoothness comes from one persistent Anime.js Animatable per scrollport —
 * continuous follow retargets `scrollTop` instead of stacking native smooth scrolls.
 */
export class PeerScrollFollow {
	activeUnit = $state.raw<ActiveUnit | null>(null);

	#clock: ComposerClock;
	#ensureMounted: EnsureMounted | undefined;
	#ensureMountedYaml: EnsureMounted | undefined;
	#getCardLengths: (() => CardListLengths | undefined) | undefined;
	#getYamlLengths: (() => CardListLengths | undefined) | undefined;
	#createAnimatableScroll: typeof createAnimatableScroll;
	#cardsEl: HTMLElement | null = null;
	#yamlEl: HTMLElement | null = null;
	#cardsAnim: AnimatableScroll | null = null;
	#yamlAnim: AnimatableScroll | null = null;

	#drivenSide: 'cards' | 'yaml' | null = null;
	#clearDriven: (() => void) | null = null;
	#scrollLeader: 'cards' | 'yaml' | null = null;
	#lastIntentSide: 'cards' | 'yaml' | null = null;
	#leaderIdleTimer: ReturnType<typeof setTimeout> | null = null;
	#peerFollowRaf: number | null = null;
	#pendingReveal: ActiveUnit | null = null;
	#disposed = false;
	/** Clock time of last paired card/YAML reorder — suppresses regen follow briefly. */
	#pairedReorderAt = Number.NEGATIVE_INFINITY;

	constructor(options?: PeerScrollFollowOptions) {
		this.#clock = options?.clock ?? browserClock();
		this.#ensureMounted = options?.ensureMounted;
		this.#ensureMountedYaml = options?.ensureMountedYaml;
		this.#getCardLengths = options?.getCardLengths;
		this.#getYamlLengths = options?.getYamlLengths ?? options?.getCardLengths;
		this.#createAnimatableScroll = options?.createAnimatableScroll ?? createAnimatableScroll;
	}

	/** Shared persisted preference (rune-sync / localStorage). */
	get enabled(): boolean {
		return scrollFollowPreference.enabled;
	}

	setEnabled(next: boolean, editingIndex?: number) {
		if (this.#disposed) return;
		scrollFollowPreference.enabled = next;
		if (!next) {
			this.activeUnit = null;
			return;
		}
		if (editingIndex !== undefined) {
			this.#setActiveUnit({ section: 'steps', index: editingIndex });
			this.#followPeerFromCards('discrete');
		}
	}

	/** Attachment for the cards scrollport (Column). Safe while follow is off — handlers no-op. */
	cardsAttach: Attachment = (node) => this.#bindScrollport('cards', node);

	/**
	 * Attachment for the YAML preview scroller (virtual step blocks + static header/follow-ups).
	 * Resolves active unit from `data-yaml-section` / `data-yaml-index` (index peer-follow).
	 */
	yamlAttach: Attachment = (node) => this.#bindScrollport('yaml', node);

	/**
	 * Shared scrollport binding for cards ↔ YAML.
	 * Cards flush a queued reveal on attach; YAML clears activeUnit when no unit is in view.
	 * Leadership / start-band vs center live elsewhere (ADR-0001).
	 */
	#bindScrollport(side: 'cards' | 'yaml', node: Element): (() => void) | void {
		const el = node as HTMLElement;
		if (side === 'cards') {
			this.#cardsEl = el;
			this.#cardsAnim?.dispose();
			this.#cardsAnim = this.#createAnimatableScroll(el);
		} else {
			this.#yamlEl = el;
			this.#yamlAnim?.dispose();
			this.#yamlAnim = this.#createAnimatableScroll(el);
		}

		const pane = side === 'cards' ? CARD_PANE : YAML_PANE;
		const getLengths = side === 'cards' ? this.#getCardLengths : this.#getYamlLengths;

		const onUserIntent = () => {
			if (!this.enabled) return;
			if (this.#drivenSide === side) this.#clearDriven?.();
			this.#lastIntentSide = side;
			this.#claimScrollLeader(side);
		};

		const onScroll = () => {
			if (!this.enabled) return;
			if (this.#drivenSide === side) return;
			const otherSide = side === 'cards' ? 'yaml' : 'cards';
			if (this.#scrollLeader === otherSide) return;
			if (this.#scrollLeader !== side && this.#lastIntentSide !== side) return;
			this.#claimScrollLeader(side);
			// Cards→YAML continuous follow: topmost intersecting card (unequal heights).
			// YAML→cards keeps center + hysteresis via resolveViewportUnit.
			const next =
				side === 'cards'
					? resolveTopmostVisibleUnit(el, pane, getLengths?.())
					: resolveViewportUnit(el, this.activeUnit, pane, getLengths?.());
			if (!next) {
				if (side === 'yaml') this.#setActiveUnit(null);
				return;
			}
			if (sameUnit(this.activeUnit, next)) return;
			this.#setActiveUnit(next);
			this.#schedulePeerFollow(side);
		};

		el.addEventListener('wheel', onUserIntent, { passive: true });
		el.addEventListener('touchstart', onUserIntent, { passive: true });
		el.addEventListener('pointerdown', onUserIntent, { passive: true });
		el.addEventListener('scroll', onScroll, { passive: true });

		if (side === 'cards') {
			const pending = this.#pendingReveal;
			if (pending) {
				this.#pendingReveal = null;
				this.#flushReveal(pending);
			}
		}

		return () => {
			el.removeEventListener('wheel', onUserIntent);
			el.removeEventListener('touchstart', onUserIntent);
			el.removeEventListener('pointerdown', onUserIntent);
			el.removeEventListener('scroll', onScroll);
			if (side === 'cards') {
				if (this.#cardsEl === el) {
					this.#cardsEl = null;
					this.#cardsAnim?.dispose();
					this.#cardsAnim = null;
				}
			} else if (this.#yamlEl === el) {
				this.#yamlEl = null;
				this.#yamlAnim?.dispose();
				this.#yamlAnim = null;
			}
		};
	}

	onEditFocus(stepIndex: number | undefined) {
		if (this.#disposed) return;
		if (stepIndex === undefined) return;
		const next: ActiveUnit = { section: 'steps', index: stepIndex };
		if (sameUnit(this.activeUnit, next)) return;
		this.#setActiveUnit(next);
		if (this.enabled) this.#followPeerFromCards('discrete');
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
		void this.#flushRevealAsync(unit);
	}

	async #flushRevealAsync(unit: ActiveUnit) {
		const cards = this.#cardsEl;
		if (!cards) return;
		const durationMs = DISCRETE_SCROLL_DURATION_MS;
		this.#beginDriven('cards', cards, durationMs);
		const scrolled = await scrollUnitIntoView(cards, unit, 'auto', CARD_PANE, {
			align: 'center',
			focus: false,
			ensureMounted: this.#ensureMounted,
			animatableScroll: this.#cardsAnim ?? undefined,
			durationMs
		});
		if (this.#disposed) return;
		if (!scrolled) this.#clearDriven?.();
		if (!this.enabled) return;
		this.#setActiveUnit(unit);
		this.#followPeerFromCards('discrete');
	}

	/**
	 * Call before a paired reorder so YAML regen (`onYamlTextChanged`) does not
	 * schedule a discrete follow that fights the scroll pin.
	 */
	notePairedReorder(): void {
		if (this.#disposed) return;
		this.#pairedReorderAt = this.#clock.now();
	}

	/**
	 * After paired reorder: if the moved unit left either viewport (e.g. shift-up
	 * from the top of the visible list), scroll that pane with `nearest` only.
	 * Mid-list swaps that stay fully visible no-op — preserves the scroll pin.
	 * Runs even when scroll-follow is off; does not use Animatable (avoids fighting FLIP).
	 */
	async revealUnitNearest(unit: ActiveUnit): Promise<boolean> {
		if (this.#disposed) return false;
		let scrolled = false;
		if (this.#cardsEl) {
			const did = await scrollUnitIntoView(this.#cardsEl, unit, 'auto', CARD_PANE, {
				align: 'nearest',
				focus: false,
				ensureMounted: this.#ensureMounted
			});
			scrolled = did || scrolled;
		}
		if (this.#yamlEl) {
			const did = await scrollUnitIntoView(this.#yamlEl, unit, 'auto', YAML_PANE, {
				align: 'nearest',
				focus: false,
				ensureMounted: this.#ensureMountedYaml
			});
			scrolled = did || scrolled;
		}
		if (scrolled) this.#setActiveUnit(unit);
		return scrolled;
	}

	/** Debounced re-follow from cards; returns cancel cleanup. */
	onYamlTextChanged(): () => void {
		if (this.#disposed || !this.enabled) return () => {};
		const now = this.#clock.now();
		if (now - this.#pairedReorderAt < PAIRED_REORDER_SUPPRESS_MS) {
			return () => {};
		}
		const timer = this.#clock.setTimeout(() => {
			if (!this.activeUnit) return;
			this.#followPeerFromCards('discrete');
		}, YAML_REGEN_DEBOUNCE_MS);
		return () => this.#clock.clearTimeout(timer);
	}

	/** After view pins selection from a card or YAML click. */
	followUnit(unit: ActiveUnit, from: 'cards' | 'yaml') {
		if (this.#disposed) return;
		this.#setActiveUnit(unit);
		if (!this.enabled) return;
		if (from === 'yaml') {
			this.#lastIntentSide = 'yaml';
			this.#claimScrollLeader('yaml');
			this.#followPeerFromYaml('discrete');
			return;
		}
		this.#lastIntentSide = 'cards';
		this.#claimScrollLeader('cards');
		this.#followPeerFromCards('discrete');
	}

	dispose() {
		if (this.#disposed) return;
		this.#disposed = true;
		this.#clearDriven?.();
		this.#clearDriven = null;
		this.#drivenSide = null;
		if (this.#leaderIdleTimer) {
			this.#clock.clearTimeout(this.#leaderIdleTimer);
			this.#leaderIdleTimer = null;
		}
		if (this.#peerFollowRaf != null) {
			this.#clock.cancelRaf(this.#peerFollowRaf);
			this.#peerFollowRaf = null;
		}
		this.#cardsAnim?.dispose();
		this.#cardsAnim = null;
		this.#yamlAnim?.dispose();
		this.#yamlAnim = null;
		this.#scrollLeader = null;
		this.#lastIntentSide = null;
		this.#pendingReveal = null;
		this.#cardsEl = null;
		this.#yamlEl = null;
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

	#beginDriven(side: 'cards' | 'yaml', el: HTMLElement, durationMs: number) {
		this.#clearDriven?.();
		this.#drivenSide = side;
		const leader: 'cards' | 'yaml' = side === 'cards' ? 'yaml' : 'cards';
		this.#claimScrollLeader(leader);
		// Mute peer scroll handlers until Animatable settles (scrollend or duration+pad).
		this.#clearDriven = watchDrivenScroll(
			el,
			'auto',
			() => {
				if (this.#drivenSide === side) this.#drivenSide = null;
				this.#clearDriven = null;
				this.#claimScrollLeader(leader);
			},
			this.#clock,
			{ idleTimeoutMs: animatableDrivenIdleMs(durationMs) }
		);
	}

	#followPeerFromCards(kind: PeerScrollKind) {
		void this.#followPeerFromCardsAsync(kind);
	}

	async #followPeerFromCardsAsync(kind: PeerScrollKind) {
		if (this.#scrollLeader === 'yaml') return;
		const unit = this.activeUnit;
		const yaml = this.#yamlEl;
		if (!unit || !yaml) return;
		const durationMs = durationFor(kind);
		this.#beginDriven('yaml', yaml, durationMs);
		// Discrete click/reveal centers like yaml→cards; continuous scroll keeps start-band.
		// Do not focus YAML on discrete — card action buttons (edit etc.) bubble through the
		// click handler and must keep focus for the form.
		const align =
			kind === 'discrete' ? 'center' : this.#scrollLeader === 'cards' ? 'start-band' : 'start';
		const scrolled = await scrollUnitIntoView(yaml, unit, 'auto', YAML_PANE, {
			align,
			focus: false,
			ensureMounted: this.#ensureMountedYaml,
			animatableScroll: this.#yamlAnim ?? undefined,
			durationMs
		});
		if (this.#disposed) return;
		if (!scrolled) {
			this.#clearDriven?.();
		}
	}

	#followPeerFromYaml(kind: PeerScrollKind) {
		void this.#followPeerFromYamlAsync(kind);
	}

	async #followPeerFromYamlAsync(kind: PeerScrollKind) {
		const unit = this.activeUnit;
		const cards = this.#cardsEl;
		if (!unit || !cards) return;
		const durationMs = durationFor(kind);
		this.#beginDriven('cards', cards, durationMs);
		const scrolled = await scrollUnitIntoView(cards, unit, 'auto', CARD_PANE, {
			align: 'center',
			focus: kind === 'discrete',
			ensureMounted: this.#ensureMounted,
			animatableScroll: this.#cardsAnim ?? undefined,
			durationMs
		});
		if (this.#disposed) return;
		if (!scrolled) {
			this.#clearDriven?.();
		}
	}

	/**
	 * Continuous scroll sync: Animatable retarget (~FOLLOW_SCROLL_DURATION_MS).
	 * Native `behavior: 'smooth'` is avoided — stacked CSS smooth scrolls chop mid-flight.
	 */
	#schedulePeerFollow(from: 'cards' | 'yaml') {
		if (this.#peerFollowRaf != null) this.#clock.cancelRaf(this.#peerFollowRaf);
		this.#peerFollowRaf = this.#clock.raf(() => {
			this.#peerFollowRaf = null;
			if (from === 'cards') this.#followPeerFromCards('follow');
			else this.#followPeerFromYaml('follow');
		});
	}
}
