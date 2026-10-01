// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';
import type { SelectedDevice, SelectedVersion } from '$pipeline-form/execution-target/types.js';

import type { WalletActionsResponse } from '@/pocketbase/types';

/** Catalog-backed wallet action (action_id path). */
export type StoredWalletActionStepData = {
	kind: 'stored';
	wallet: HubItem;
	version: SelectedVersion;
	device: SelectedDevice;
	action: WalletActionsResponse;
	parameters?: { [key: string]: string };
};

/**
 * Inline Maestro payload (action_code path).
 * Minimal shape for board display + YAML round-trip until a full editor exists.
 */
export type InlineWalletActionStepData = {
	kind: 'inline';
	actionCode: string;
	version: SelectedVersion;
	device: SelectedDevice;
	parameters?: { [key: string]: string };
};

export type WalletActionStepData = StoredWalletActionStepData | InlineWalletActionStepData;

export function isInlineWalletActionStepData(value: unknown): value is InlineWalletActionStepData {
	if (!value || typeof value !== 'object') return false;
	return (value as { kind?: unknown }).kind === 'inline';
}

export function isStoredWalletActionStepData(value: unknown): value is StoredWalletActionStepData {
	if (!value || typeof value !== 'object') return false;
	return (value as { kind?: unknown }).kind === 'stored';
}
