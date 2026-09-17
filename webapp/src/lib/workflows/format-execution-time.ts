// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import dayjs from 'dayjs';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(timezone);

export type SplitExecutionTimes = {
	/** Start calendar date as `DD/MM/YYYY`. */
	date: string;
	/** Start clock as `HH:mm`. */
	start: string;
	/** End clock as `HH:mm`, when an end instant exists. */
	end?: string;
	/**
	 * Calendar-day offset of end relative to start in the display timezone.
	 * `0` same day, `1` next day, etc.
	 */
	endDayOffset: number;
};

function parseInZone(value: string | undefined | null, timeZone?: string) {
	if (!value) return undefined;
	const parsed = dayjs(value);
	if (!parsed.isValid()) return undefined;
	return timeZone ? parsed.tz(timeZone) : parsed.local();
}

/** Formats an instant as Credimi date `DD/MM/YYYY` in the user timezone. */
export function formatExecutionDate(
	value: string | undefined | null,
	timeZone?: string
): string | undefined {
	return parseInZone(value, timeZone)?.format('DD/MM/YYYY');
}

/** Formats an instant as clock `HH:mm` in the user timezone. */
export function formatExecutionClock(
	value: string | undefined | null,
	timeZone?: string
): string | undefined {
	return parseInZone(value, timeZone)?.format('HH:mm');
}

/**
 * Splits start/end instants into date + clocks for dense tables.
 * Marks overnight (and multi-day) ends via `endDayOffset`.
 */
export function splitExecutionTimes(
	startTime: string | undefined | null,
	endTime: string | undefined | null,
	timeZone?: string
): SplitExecutionTimes | undefined {
	const start = parseInZone(startTime, timeZone);
	if (!start) return undefined;

	const end = parseInZone(endTime, timeZone);
	let endDayOffset = 0;
	if (end) {
		const startDay = start.startOf('day');
		const endDay = end.startOf('day');
		endDayOffset = Math.max(0, endDay.diff(startDay, 'day'));
	}

	return {
		date: start.format('DD/MM/YYYY'),
		start: start.format('HH:mm'),
		end: end?.format('HH:mm'),
		endDayOffset
	};
}

/** Renders end clock with a next-day / multi-day marker when needed. */
export function formatEndClock(parts: Pick<SplitExecutionTimes, 'end' | 'endDayOffset'>): string | undefined {
	if (!parts.end) return undefined;
	if (parts.endDayOffset <= 0) return parts.end;
	if (parts.endDayOffset === 1) return `+${parts.end}`;
	return `+${parts.endDayOffset}d ${parts.end}`;
}
