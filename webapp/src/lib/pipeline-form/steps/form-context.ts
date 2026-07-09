// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ExecutionTarget } from '../execution-target/types.js';
import type { FormIntent } from './types.js';

export type ExecutionTargetFormContext = {
	getExecutionTarget: () => ExecutionTarget | undefined;
	isExecutionTargetLocked: () => boolean;
};

export type InitFormOptions<T> = {
	intent?: FormIntent;
	initial?: T;
} & Partial<ExecutionTargetFormContext>;
