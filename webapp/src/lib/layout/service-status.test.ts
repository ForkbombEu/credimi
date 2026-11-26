// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { expect, test } from 'vitest';

import { overallStatus } from './service-status';

test('reads the latest heartbeat of every monitor', () => {
	expect(overallStatus({ a: [{ status: 0 }, { status: 1 }], b: [{ status: 3 }] })).toBe('up');
	expect(overallStatus({ a: [{ status: 1 }], b: [{ status: 1 }, { status: 0 }] })).toBe(
		'degraded'
	);
	expect(overallStatus({ a: [{ status: 1 }], b: [] })).toBe('unknown');
	expect(overallStatus(undefined)).toBe('unknown');
});
