// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { formatTranscriptTime } from './transcript-time';

describe('formatTranscriptTime', () => {
	it('formats the time as HH:mm in the time zone', () => {
		expect(formatTranscriptTime('2026-01-02T03:04:05Z', 'UTC')).toBe('03:04');
		expect(formatTranscriptTime('2026-01-02T23:59:00Z', 'Europe/Rome')).toBe('00:59');
	});

	it('returns an empty string for an invalid time', () => {
		expect(formatTranscriptTime('not a time', 'UTC')).toBe('');
		expect(formatTranscriptTime('', 'UTC')).toBe('');
	});
});
