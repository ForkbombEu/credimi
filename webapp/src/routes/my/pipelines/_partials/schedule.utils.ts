// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

import type { SelectOption } from '@/components/ui-custom/utils';

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

export const scheduleModeOptions: SelectOption<ScheduleModeName>[] = [
	{
		label: m.daily(),
		value: 'daily'
	},
	{
		label: m.weekly(),
		value: 'weekly'
	},
	{
		label: m.monthly(),
		value: 'monthly'
	}
];
