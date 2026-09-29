// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { getExecutionDevices, type ExecutionSummary } from './workflows';

describe('getExecutionDevices', () => {
	it('prefers structured devices from the API', () => {
		const summary = {
			devices: [
				{ device_id: 'org/runner/a', name: 'a', live_view: true },
				{ device_id: 'org/runner/b', name: 'b', live_view: false }
			]
		} as ExecutionSummary;
		expect(getExecutionDevices(summary)).toEqual(summary.devices);
	});

	it('falls back to legacy device_records without live view', () => {
		const summary = {
			device_records: [{ name: 'legacy' }]
		} as ExecutionSummary;
		expect(getExecutionDevices(summary)).toEqual([
			{ device_id: 'legacy', name: 'legacy', live_view: false }
		]);
	});

	it('returns empty when neither field is present', () => {
		expect(getExecutionDevices({} as ExecutionSummary)).toEqual([]);
	});
});
