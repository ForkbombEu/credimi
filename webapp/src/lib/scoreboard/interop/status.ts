// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InteropStatus } from './types';

const STATUS_STYLES: Record<InteropStatus, { bg: string; text: string; dot: string }> = {
	stable: {
		bg: 'bg-emerald-100',
		text: 'text-emerald-800',
		dot: 'bg-emerald-500'
	},
	flaky: {
		bg: 'bg-amber-100',
		text: 'text-amber-800',
		dot: 'bg-amber-500'
	},
	failing: {
		bg: 'bg-orange-100',
		text: 'text-orange-800',
		dot: 'bg-orange-500'
	},
	broken: {
		bg: 'bg-red-100',
		text: 'text-red-800',
		dot: 'bg-red-500'
	}
};

export function interopStatusStyles(status: InteropStatus) {
	return STATUS_STYLES[status];
}
