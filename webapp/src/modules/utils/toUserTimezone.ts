// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import { currentUser, pb } from '@/pocketbase';
import { get } from 'svelte/store';

dayjs.extend(utc);
dayjs.extend(timezone);

export const toUserTimezone = (date: string | undefined): string | undefined => {
	if (!date) {
		return undefined;
	}
	const userTimezone = get(currentUser)?.Timezone;
	if (!userTimezone) {
		return date.toString();
	}

	const parsedDate = dayjs(date);
	if (!parsedDate.isValid()) {
		throw new Error('Invalid date provided');
	}

	return parsedDate.tz(userTimezone).format();
};
