// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { types as t } from '#';
import { Task, fromPromise, reject, resolve } from 'true-myth/task';

//

export { fromPromise, reject, resolve, type Task };

export type WithError<T> = Task<T, t.BaseError>;

export function withError<T>(promise: Promise<T>, Error = t.BaseError): WithError<T> {
	return fromPromise(promise, (e) => new Error(e));
}

export async function run<T, E>(task: Task<T, E>): Promise<T> {
	const v = await task;
	if (v.isOk) return v.value;
	throw v.error;
}
