// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Formats an execution timestamp as `DD MMM YYYY, HH:mm` (Credimi brand format). */
export function formatExecutionTimestamp(
	value: string | undefined | null,
	timeZone?: string
): string | undefined {
	if (!value) return undefined;

	const parsed = new Date(value);
	if (Number.isNaN(parsed.getTime())) return undefined;

	return parsed.toLocaleString(undefined, {
		year: 'numeric',
		month: 'short',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		hour12: false,
		...(timeZone ? { timeZone } : {})
	});
}
