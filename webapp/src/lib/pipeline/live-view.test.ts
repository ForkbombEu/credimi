// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { getExecutionDevices, type ExecutionSummary } from './workflows';

describe('getExecutionDevices', () => {
	it('returns structured devices from the API', () => {
		const summary = {
			devices: [
				{ device_id: 'org/runner/a', name: 'a', live_view: true },
				{ device_id: 'org/runner/b', name: 'b', live_view: false }
			]
		} as ExecutionSummary;
		expect(getExecutionDevices(summary)).toEqual(summary.devices);
	});

	it('returns empty when devices are absent', () => {
		expect(getExecutionDevices({} as ExecutionSummary)).toEqual([]);
	});
});
