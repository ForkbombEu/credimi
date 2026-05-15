// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ScoreboardRow } from '../types';

import { emptyExecutionStats, fromScoreboardRow } from './from-scoreboard-row';

describe('fromScoreboardRow', () => {
	it('returns undefined when row is undefined', () => {
		expect(fromScoreboardRow(undefined)).toBeUndefined();
	});

	it('maps scoreboard fields with defaults', () => {
		const row = {
			total_runs: 10,
			total_successes: 8,
			success_rate: 80,
			manually_executed_runs: 3,
			scheduled_runs: 2,
			CI_runs: 5
		} as ScoreboardRow;

		expect(fromScoreboardRow(row)).toEqual({
			total: 10,
			successes: 8,
			percent: 80,
			manual: 3,
			scheduled: 2,
			ci: 5
		});
	});

	it('coalesces missing optional numbers to zero', () => {
		const row = { total_runs: 0 } as ScoreboardRow;
		expect(fromScoreboardRow(row)).toEqual({
			total: 0,
			successes: 0,
			percent: 0,
			manual: 0,
			scheduled: 0,
			ci: 0
		});
	});
});

describe('emptyExecutionStats', () => {
	it('is all zeros', () => {
		expect(emptyExecutionStats).toEqual({
			total: 0,
			successes: 0,
			percent: 0,
			manual: 0,
			scheduled: 0,
			ci: 0
		});
	});
});
