// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InitFormOptions, FormIntent } from './types.js';

type CreateInitFormOptionsInput<T> = {
	intent: FormIntent;
	initial?: T;
	getExecutionTarget?: InitFormOptions<T>['getExecutionTarget'];
	isExecutionTargetLocked?: InitFormOptions<T>['isExecutionTargetLocked'];
};

export function createInitFormOptions<T>(opts: CreateInitFormOptionsInput<T>): InitFormOptions<T> {
	return {
		getExecutionTarget: () => undefined,
		isExecutionTargetLocked: () => false,
		...opts
	};
}
