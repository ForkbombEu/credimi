// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Task, fromPromise } from 'true-myth/task';

import { types as t } from '@/v2';

//

export { fromPromise, type Task };

export type WithError<T> = Task<T, t.BaseError>;

export function withError<T>(promise: Promise<T>): WithError<T> {
	return fromPromise(promise, (e) => new t.BaseError(e));
}

export async function run<T, E>(task: Task<T, E>): Promise<T> {
	const v = await task;
	if (v.isOk) return v.value;
	throw v.error;
}
