// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Fill/unlock/selection chrome shared by StepCard and FollowUpCard — face stays out. */
export type InCardHostChrome = {
	editing?: boolean;
	expandReady?: boolean;
	selected?: boolean;
	hovered?: boolean;
	faded?: boolean;
	cardFillMaxPx?: number;
	onExitUnlock?: () => void;
};
