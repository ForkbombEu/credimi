// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Record } from 'effect';
import { z } from 'zod';

import type { SelectOption } from '@/components/ui-custom/utils';
import type { SchedulesResponse } from '@/pocketbase/types';

import { m } from '@/i18n';

//

export const dayOptions: SelectOption<number>[] = [
	{ label: m.monday(), value: 0 },
	{ label: m.tuesday(), value: 1 },
	{ label: m.wednesday(), value: 2 },
	{ label: m.thursday(), value: 3 },
	{ label: m.friday(), value: 4 },
	{ label: m.saturday(), value: 5 },
	{ label: m.sunday(), value: 6 }
];

export function getDayLabel(day?: number) {
	if (day === undefined) return undefined;
	return dayOptions.find((option) => option.value === day)?.label;
}

//

export const scheduleModeSchema = z.union([
	z.object({
		mode: z.literal('daily')
	}),
	z.object({
		mode: z.literal('weekly'),
		day: z.number().min(0).max(6)
	}),
	z.object({
		mode: z.literal('monthly'),
		day: z.number().min(1).max(31)
	})
]);

export type ScheduleMode = z.infer<typeof scheduleModeSchema>;

export type ScheduleModeName = ScheduleMode['mode'];

const scheduleModeNameTranslations: Record<ScheduleModeName, string> = {
	daily: m.daily(),
	weekly: m.weekly(),
	monthly: m.monthly()
};

export const scheduleModeOptions: SelectOption<ScheduleModeName>[] = Record.toEntries(
	scheduleModeNameTranslations
).map(([name, label]) => ({
	label,
	value: name
}));

//

export function scheduleModeLabel(mode: ScheduleMode) {
	if (mode.mode === 'daily') {
		return m.daily();
	} else if (mode.mode === 'weekly') {
		return m.weekly() + ' (' + getDayLabel(mode.day) + ')';
	} else {
		return m.monthly() + ' (' + mode.day + ')';
	}
}

//

type ScheduleStatus = {
	display_name: string;
	next_action_time: string;
	paused: boolean;
};

export type EnrichedSchedule = SchedulesResponse & {
	__schedule_status__: ScheduleStatus;
};
