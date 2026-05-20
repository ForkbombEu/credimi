// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RunnerRecord } from './types';

export function filterRunners(runners: readonly RunnerRecord[], text: string): RunnerRecord[] {
	const search = text.trim().toLowerCase();
	if (!search) return [...runners];

	return runners.filter(
		(runner) =>
			runner.name.toLowerCase().includes(search) || runner.path.toLowerCase().includes(search)
	);
}
