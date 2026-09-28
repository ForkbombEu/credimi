// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ExecutionSummary } from './workflows';

import { liveViewDeviceTargets } from './live-view';

function summary(status: ExecutionSummary['status'], deviceIds?: string[]): ExecutionSummary {
	return { status, device_ids: deviceIds } as ExecutionSummary;
}

describe('liveViewDeviceTargets', () => {
	it('returns no targets when the execution is not running', () => {
		expect(liveViewDeviceTargets(summary('Completed', ['tenant/runner/device-a']))).toEqual([]);
	});

	it('returns no targets for a running execution without devices', () => {
		expect(liveViewDeviceTargets(summary('Running'))).toEqual([]);
	});

	it('omits the label for a single device', () => {
		expect(liveViewDeviceTargets(summary('Running', ['tenant/runner/device-a']))).toEqual([
			{ deviceId: 'tenant/runner/device-a', deviceLabel: null }
		]);
	});

	it('labels each device when there are several', () => {
		expect(
			liveViewDeviceTargets(
				summary('Running', ['tenant/runner/device-a', 'tenant/runner/device-b'])
			)
		).toEqual([
			{ deviceId: 'tenant/runner/device-a', deviceLabel: 'device-a' },
			{ deviceId: 'tenant/runner/device-b', deviceLabel: 'device-b' }
		]);
	});
});
