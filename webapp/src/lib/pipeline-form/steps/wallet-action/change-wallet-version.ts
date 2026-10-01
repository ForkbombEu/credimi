// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SelectedVersion } from '$pipeline-form/execution-target/types.js';

import type { StoredWalletActionStepData } from './types.js';

import { isStoredWalletActionStepData } from './types.js';

/**
 * Whether enriched wallet-action step data can participate in Matching mobile steps
 * (Change wallet version). Inline steps are excluded.
 */
export function isMatchingMobileStepData(data: unknown): data is StoredWalletActionStepData {
	return isStoredWalletActionStepData(data);
}

/**
 * Apply a bulk Step version when this enriched step participates in Change wallet version
 * for `walletId`. Returns undefined when the step should be left unchanged (inline,
 * other wallet, or non-stored shape).
 */
export function applyWalletActionStepVersion(
	data: unknown,
	walletId: string,
	version: SelectedVersion
): StoredWalletActionStepData | undefined {
	if (!isMatchingMobileStepData(data)) return undefined;
	if (data.wallet.id !== walletId) return undefined;
	return { ...data, version };
}
