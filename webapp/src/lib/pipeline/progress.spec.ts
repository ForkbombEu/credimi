// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { formatDurationParts } from './progress';

describe('formatDurationParts', () => {
	it('formats seconds only', () => {
		expect(formatDurationParts(45)).toBe('45s');
	});

	it('formats minutes and seconds', () => {
		expect(formatDurationParts(133)).toBe('2m 13s');
	});

	it('formats hours and minutes', () => {
		expect(formatDurationParts(3725)).toBe('1h 2m');
	});

	it('clamps negative values to zero', () => {
		expect(formatDurationParts(-10)).toBe('0s');
	});
});
