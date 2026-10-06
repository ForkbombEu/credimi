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

export type PeerScrollKind = 'follow' | 'discrete';

export type PeerScrollFollowOptions = {
	clock?: ComposerClock;
	ensureMounted?: EnsureMounted;
	ensureMountedYaml?: EnsureMounted;
	getCardLengths?: () => CardListLengths | undefined;
	getYamlLengths?: () => CardListLengths | undefined;
	createAnimatableScroll?: typeof createAnimatableScroll;
	isCardsParked?: () => boolean;
};

export const YAML_REGEN_DEBOUNCE_MS = 130;
/** Slack past debounce so paired-reorder regen does not schedule a competing follow. */
const PAIRED_REORDER_SUPPRESS_MS = YAML_REGEN_DEBOUNCE_MS + 50;
const LEADER_IDLE_MS = 900;

function durationFor(kind: PeerScrollKind): number {
	return kind === 'follow' ? FOLLOW_SCROLL_DURATION_MS : DISCRETE_SCROLL_DURATION_MS;
}

/**
 * Does not select or highlight a step (see UnitHighlight).
 * Continuous follow retargets Animatable `scrollTop` instead of stacking native smooth scrolls.
 */
export class PeerScrollFollow {
	activeUnit = $state.raw<ActiveUnit | null>(null);

	#clock: ComposerClock;
	#ensureMounted: EnsureMounted | undefined;
	#ensureMountedYaml: EnsureMounted | undefined;
	#getCardLengths: (() => CardListLengths | undefined) | undefined;
	#getYamlLengths: (() => CardListLengths | undefined) | undefined;
	#createAnimatableScroll: typeof createAnimatableScroll;
	#isCardsParked: (() => boolean) | undefined;
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
	/**
	 * In-card enter hard start: mute peer sync so YAML start-align cannot
	 * center-yank cards (and clear cards driven) before expand/park.
	 */
	#editHardStart = false;
	/** Bumps each onEditFocus so a deferred mute-clear cannot drop a newer enter. */
	#editHardStartGen = 0;
	#pairedReorderAt = Number.NEGATIVE_INFINITY;

	constructor(options?: PeerScrollFollowOptions) {
		this.#clock = options?.clock ?? browserClock();
		this.#ensureMounted = options?.ensureMounted;
		this.#ensureMountedYaml = options?.ensureMountedYaml;
		this.#getCardLengths = options?.getCardLengths;
		this.#getYamlLengths = options?.getYamlLengths ?? options?.getCardLengths;
		this.#createAnimatableScroll = options?.createAnimatableScroll ?? createAnimatableScroll;
		this.#isCardsParked = options?.isCardsParked;
	}

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
			if (!this.#isCardsParked?.()) this.#followPeerFromCards('discrete');
		}
	}

	cardsAttach: Attachment = (node) => this.#bindScrollport('cards', node);

	yamlAttach: Attachment = (node) => this.#bindScrollport('yaml', node);

	/** Leadership / start-band vs center live elsewhere (ADR-0001). */
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
			if (this.#isCardsParked?.()) return;
			if (this.#drivenSide === side) this.#clearDriven?.();
			this.#lastIntentSide = side;
			this.#claimScrollLeader(side);
		};

		const onScroll = () => {
			if (!this.enabled) return;
			if (this.#isCardsParked?.()) return;
			if (this.#editHardStart) return;
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

	onEditFocus(unit: ActiveUnit): Promise<void> {
		if (this.#disposed) return Promise.resolve();
		const hardStartGen = ++this.#editHardStartGen;
		this.#editHardStart = true;
		this.#setActiveUnit(unit);
		this.#lastIntentSide = 'cards';
		this.#claimScrollLeader('cards');
		void this.#startAlignYamlForEdit(unit);
		return this.#startAlignCardsForEdit(unit)
			.then(() => {
				if (this.#disposed) return;
				this.#snapCardsStartAlign(unit);
				// Keep cards as leader until Twin-pane parks (next microtask).
				this.#claimScrollLeader('cards');
			})
			.finally(() => {
				// Clear after awaiter parks so a trailing YAML scroll cannot peer-yank.
				this.#clock.setTimeout(() => {
					if (this.#editHardStartGen === hardStartGen) this.#editHardStart = false;
				}, 0);
			});
	}

	async #startAlignCardsForEdit(unit: ActiveUnit): Promise<void> {
		const cards = this.#cardsEl;
		if (!cards) return;
		const durationMs = DISCRETE_SCROLL_DURATION_MS;
		const settled = this.#beginDriven('cards', cards, durationMs);
		// beginDriven claims the opposite pane as leader; keep cards for hard start
		// so YAML scroll cannot become the continuous-follow source mid-enter.
		this.#claimScrollLeader('cards');
		const scrolled = await scrollUnitIntoView(cards, unit, 'auto', CARD_PANE, {
			align: 'start',
			focus: false,
			ensureMounted: this.#ensureMounted,
			animatableScroll: this.#cardsAnim ?? undefined,
			durationMs
		});
		if (this.#disposed) return;
		if (!scrolled) {
			this.#clearDriven?.();
			this.#claimScrollLeader('cards');
			return;
		}
		await settled;
		this.#claimScrollLeader('cards');
	}

	/** Instant start-align so the card top is not still clipped. */
	#snapCardsStartAlign(unit: ActiveUnit): void {
		const cards = this.#cardsEl;
		if (!cards) return;
		void scrollUnitIntoView(cards, unit, 'auto', CARD_PANE, {
			align: 'start',
			focus: false,
			animatableScroll: this.#cardsAnim ?? undefined,
			durationMs: 0
		});
	}

	async #startAlignYamlForEdit(unit: ActiveUnit): Promise<void> {
		const yaml = this.#yamlEl;
		if (!yaml) return;
		const durationMs = DISCRETE_SCROLL_DURATION_MS;
		await scrollUnitIntoView(yaml, unit, 'auto', YAML_PANE, {
			align: 'start',
			focus: false,
			ensureMounted: this.#ensureMountedYaml,
			animatableScroll: this.#yamlAnim ?? undefined,
			durationMs
		});
	}

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
		void this.#beginDriven('cards', cards, durationMs);
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

	onYamlTextChanged(): () => void {
		if (this.#disposed || !this.enabled || this.#isCardsParked?.()) return () => {};
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

	followUnit(unit: ActiveUnit, from: 'cards' | 'yaml') {
		if (this.#disposed) return;
		this.#setActiveUnit(unit);
		if (!this.enabled) return;
		if (this.#isCardsParked?.()) return;
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
		this.#editHardStart = false;
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

	#beginDriven(side: 'cards' | 'yaml', el: HTMLElement, durationMs: number): Promise<void> {
		this.#clearDriven?.();
		this.#drivenSide = side;
		const leader: 'cards' | 'yaml' = side === 'cards' ? 'yaml' : 'cards';
		this.#claimScrollLeader(leader);
		return new Promise((resolve) => {
			this.#clearDriven = watchDrivenScroll(
				el,
				'auto',
				() => {
					if (this.#drivenSide === side) this.#drivenSide = null;
					this.#clearDriven = null;
					this.#claimScrollLeader(leader);
					resolve();
				},
				this.#clock,
				{ idleTimeoutMs: animatableDrivenIdleMs(durationMs) }
			);
		});
	}

	#followPeerFromCards(kind: PeerScrollKind) {
		if (this.#isCardsParked?.()) return;
		void this.#followPeerFromCardsAsync(kind);
	}

	async #followPeerFromCardsAsync(kind: PeerScrollKind) {
		if (this.#isCardsParked?.()) return;
		if (this.#scrollLeader === 'yaml') return;
		const unit = this.activeUnit;
		const yaml = this.#yamlEl;
		if (!unit || !yaml) return;
		const durationMs = durationFor(kind);
		void this.#beginDriven('yaml', yaml, durationMs);
		// Discrete click/reveal centers like yaml→cards; continuous scroll keeps start-band.
		// Do not focus YAML on discrete — card action buttons (edit etc.) bubble through the
		// click handler and must keep focus for the form.
		const align =
			kind === 'discrete'
				? 'center'
				: this.#scrollLeader === 'cards'
					? 'start-band'
					: 'start';
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
		if (this.#isCardsParked?.()) return;
		void this.#followPeerFromYamlAsync(kind);
	}

	async #followPeerFromYamlAsync(kind: PeerScrollKind) {
		if (this.#isCardsParked?.()) return;
		const unit = this.activeUnit;
		const cards = this.#cardsEl;
		if (!unit || !cards) return;
		const durationMs = durationFor(kind);
		void this.#beginDriven('cards', cards, durationMs);
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
