// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';
import type { Record } from '$lib/pipeline/runner';

import type { WalletVersionsResponse } from '@/pocketbase/types';

//

export const GLOBAL_RUNNER = 'global' as const;
export const EXTERNAL_VERSION = 'installed_from_external_source' as const;

export type SelectedRunner = Record | typeof GLOBAL_RUNNER;
export type SelectedVersion = WalletVersionsResponse | typeof EXTERNAL_VERSION;

export type MobileTargetFields = {
	wallet: HubItem;
	version: SelectedVersion;
	runner: SelectedRunner;
};
