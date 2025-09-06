// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as task from 'true-myth/task';

import { run } from './task';

//

export class Runner<Tasks extends task.AnyTask> {
	currentTask = $state<Tasks>();

	async run<Task extends Tasks>(task: Task): Promise<task.ResolvesTo<Task>> {
		this.currentTask = task;
		return await run(this.currentTask);
	}
}
