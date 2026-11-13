// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { AuthorDevice, BuildOption } from './api';

import {
	buildPlatforms,
	preferredDevice,
	toDeviceRecords,
	walletBuilds,
	walletPath
} from './builds';

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

	it("finds a public wallet's builds under its owner's path, not the caller's", () => {
		const builds = [
			build('org/w', ['android']),
			build('other/w', ['android']),
			build('other/w/v1', ['android'])
		];
		const path = walletPath({ canonified_name: 'other' }, { canonified_name: 'w' });
		expect(walletBuilds(builds, path).map((b) => b.identifier)).toEqual([
			'other/w',
			'other/w/v1'
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

describe('preferredDevice', () => {
	const android = new Set(['android']);

	it('picks the online agent device with the fewest runs ahead', () => {
		const busy = { ...device('org/r/busy', 'android'), runs_ahead: 3 };
		const idle = { ...device('org/r/idle', 'android'), runs_ahead: 1 };
		expect(preferredDevice([busy, idle], android)?.path).toBe('org/r/idle');
	});

	it('skips offline devices, devices without a running agent and other platforms', () => {
		const offline = {
			...device('org/r/offline', 'android'),
			is_online: false,
			runs_ahead: 0
		};
		const noAgent = { ...device('org/r/no-agent', 'android', false), runs_ahead: 0 };
		const iphone = { ...device('org/r/iphone', 'ios'), runs_ahead: 0 };
		const pixel = { ...device('org/r/pixel', 'android'), runs_ahead: 2 };
		expect(preferredDevice([offline, noAgent, iphone, pixel], android)?.path).toBe(
			'org/r/pixel'
		);
	});

	it('returns undefined when no device qualifies', () => {
		const offline = { ...device('org/r/offline', 'android'), is_online: false };
		expect(preferredDevice([offline, device('org/r/iphone', 'ios')], android)).toBeUndefined();
	});

	it('keeps list order on ties', () => {
		const first = device('org/r/first', 'android');
		const second = { ...device('org/r/second', 'android'), runs_ahead: 0 };
		expect(preferredDevice([first, second], android)?.path).toBe('org/r/first');
	});
});
