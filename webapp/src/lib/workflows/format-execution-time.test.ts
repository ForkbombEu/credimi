// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	formatEndClock,
	formatExecutionClock,
	formatExecutionDate,
	splitExecutionTimes
} from './format-execution-time';

describe('splitExecutionTimes', () => {
	it('splits same-day start and end without a next-day marker', () => {
		const parts = splitExecutionTimes(
			'2026-09-17T12:02:11Z',
			'2026-09-17T12:07:44Z',
			'UTC'
		);
		expect(parts).toEqual({
			date: '17/09/2026',
			start: '12:02',
			end: '12:07',
			endDayOffset: 0
		});
		expect(formatEndClock(parts!)).toBe('12:07');
	});

	it('marks end on the next calendar day', () => {
		const parts = splitExecutionTimes(
			'2026-09-17T22:58:01Z',
			'2026-09-18T00:03:12Z',
			'UTC'
		);
		expect(parts).toEqual({
			date: '17/09/2026',
			start: '22:58',
			end: '00:03',
			endDayOffset: 1
		});
		expect(formatEndClock(parts!)).toBe('+00:03');
	});

	it('marks multi-day end offsets', () => {
		const parts = splitExecutionTimes(
			'2026-09-17T10:00:00Z',
			'2026-09-19T11:30:00Z',
			'UTC'
		);
		expect(parts?.endDayOffset).toBe(2);
		expect(formatEndClock(parts!)).toBe('+2d 11:30');
	});

	it('omits end when missing', () => {
		const parts = splitExecutionTimes('2026-09-17T12:02:11Z', undefined, 'UTC');
		expect(parts).toEqual({
			date: '17/09/2026',
			start: '12:02',
			end: undefined,
			endDayOffset: 0
		});
		expect(formatEndClock(parts!)).toBeUndefined();
	});

	it('applies the display timezone', () => {
		const parts = splitExecutionTimes(
			'2026-09-17T22:30:00Z',
			'2026-09-17T23:15:00Z',
			'Europe/Rome'
		);
		expect(parts?.date).toBe('18/09/2026');
		expect(parts?.start).toBe('00:30');
		expect(parts?.end).toBe('01:15');
	});
});

describe('formatExecutionDate / formatExecutionClock', () => {
	it('formats date and clock parts', () => {
		expect(formatExecutionDate('2026-09-17T12:02:11Z', 'UTC')).toBe('17/09/2026');
		expect(formatExecutionClock('2026-09-17T12:02:11Z', 'UTC')).toBe('12:02');
	});
});
