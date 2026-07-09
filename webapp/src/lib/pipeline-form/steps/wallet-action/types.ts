// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WalletActionsResponse } from '@/pocketbase/types';

import type { MobileTargetFields } from '../../shared/mobile-target.js';

export type WalletActionStepData = MobileTargetFields & { action: WalletActionsResponse };
export type { SelectedRunner, SelectedVersion } from '../../shared/mobile-target.js';
export { GLOBAL_RUNNER, EXTERNAL_VERSION } from '../../shared/mobile-target.js';
