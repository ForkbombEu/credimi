// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { AuthorDevice, BuildOption } from './api';

import { buildPlatforms, toDeviceRecords, walletBuilds } from './builds';

function build(identifier: string, platforms: string[]): BuildOption {
	return {
		identifier,
		wallet_name: 'W',
		organization_name: 'Org',
		version_tag: identifier.split('/')[2] ?? '',
		platforms,
		source: identifier.split('/').length === 2 ? 'store' : 'upload'
	};
}

function device(path: string, platform: string, agentAvailable = true): AuthorDevice {
	return {
		path,
		name: path,
		runner_name: 'runner',
		runner_owner: 'Org',
		model: 'gpt',
		type: platform === 'ios' ? 'ios_simulator' : 'android_phone',
		platform,
		is_online: true,
		agent_available: agentAvailable
	};
}

describe('walletBuilds', () => {
	it('keeps the wallet store build and its versions only', () => {
		const builds = [
			build('org/w', ['android']),
			build('org/w/v1', ['android']),
			build('org/w2/v1', ['android'])
		];
		expect(walletBuilds(builds, 'org/w').map((b) => b.identifier)).toEqual([
			'org/w',
			'org/w/v1'
		]);
	});
});

describe('toDeviceRecords', () => {
	it('drops devices of platforms without builds', () => {
		const platforms = buildPlatforms([build('org/w/v1', ['android'])]);
		const records = toDeviceRecords(
			[device('org/r/pixel', 'android'), device('org/r/iphone', 'ios')],
			platforms
		);
		expect(records.map((r) => r.path)).toEqual(['org/r/pixel']);
	});

	it('blocks devices whose agent is not running', () => {
		const records = toDeviceRecords(
			[device('org/r/pixel', 'android', false), device('org/r/other', 'android')],
			new Set(['android'])
		);
		expect(records[0].unavailableReason).toBeTruthy();
		expect(records[1].unavailableReason).toBeUndefined();
	});

	it('marks the online status as known without the device catalog', () => {
		const records = toDeviceRecords([device('org/r/pixel', 'android')], new Set(['android']));
		expect(records[0].statusKnown).toBe(true);
	});
});
