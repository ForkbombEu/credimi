// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export * from 'true-myth/task';
import type { Task } from 'true-myth/task';

//

export async function run<T, E>(task: Task<T, E>): Promise<T> {
	const v = await task;
	if (v.isOk) return v.value;
	throw v.error;
}
