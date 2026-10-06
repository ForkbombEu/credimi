// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Fill/unlock/selection chrome shared by StepCard and FollowUpCard — face stays out. */
export type InCardHostChrome = {
	editing?: boolean;
	/** Enter-ready: parent `editing && session.inCard.phase === 'still'`. */
	expandReady?: boolean;
	selected?: boolean;
	hovered?: boolean;
	/** Sibling of the card being edited in place: dimmed and non-interactive (except the pencil). */
	faded?: boolean;
	/** Max height of the card while editing, in px. Composer passes `session.cardFillMaxPx`. */
	cardFillMaxPx?: number;
	/** Paired with held-form clear — Twin-pane `noteExitComplete`. */
	onExitUnlock?: () => void;
};
