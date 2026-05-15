// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ScoreboardRow } from '../types';

export type ExecutionStats = {
	total: number;
	successes: number;
	percent: number;
	manual: number;
	scheduled: number;
	ci: number;
};

export const emptyExecutionStats: ExecutionStats = {
	total: 0,
	successes: 0,
	percent: 0,
	manual: 0,
	scheduled: 0,
	ci: 0
};

export function fromScoreboardRow(row: ScoreboardRow | undefined): ExecutionStats | undefined {
	if (!row) return undefined;

	return {
		total: row.total_runs ?? 0,
		successes: row.total_successes ?? 0,
		percent: row.success_rate ?? 0,
		manual: row.manually_executed_runs ?? 0,
		scheduled: row.scheduled_runs ?? 0,
		ci: row.CI_runs ?? 0
	};
}
