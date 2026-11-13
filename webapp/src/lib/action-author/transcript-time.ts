// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Formats a transcript entry's RFC 3339 time as `HH:mm`, empty when it is not a time. */
export function formatTranscriptTime(value: string, timeZone?: string): string {
	const parsed = new Date(value);
	if (Number.isNaN(parsed.getTime())) return '';
	return parsed.toLocaleTimeString('en-GB', {
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23',
		...(timeZone ? { timeZone } : {})
	});
}
