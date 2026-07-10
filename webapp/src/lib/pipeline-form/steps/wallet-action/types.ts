// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { isExecutionTarget, type ExecutionTarget } from '$pipeline-form/execution-target/types.js';

import type { WalletActionsResponse } from '@/pocketbase/types';

export type WalletActionStepData = ExecutionTarget & { action: WalletActionsResponse };

export function isWalletActionStepData(value: unknown): value is WalletActionStepData {
	if (!isExecutionTarget(value) || !('action' in value)) return false;
	const action = (value as WalletActionStepData).action;
	return (
		!!action && typeof action === 'object' && 'id' in action && typeof action.id === 'string'
	);
}
