// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { FcafCoverageStatus, FcafRoleCoverage, FcafRoleProgress, FcafRoleStatus } from './api';

import {
	coverageCounts,
	coverageLabel,
	coverageVariant,
	coverageVerdict,
	modePlatforms,
	percent,
	progressDone,
	replayStatusLabel,
	replayStatusVariant,
	roleStatusLabel,
	roleStatusVariant
} from './fcaf';

function progress(role: string, status: FcafRoleStatus): FcafRoleProgress {
	return { role, status, key: '', note: '' };
}

function coverage(
	role: string,
	status: FcafCoverageStatus,
	result: FcafRoleCoverage['result'] = ''
): FcafRoleCoverage {
	return {
		role,
		summary: '',
		outcome: '',
		category: '',
		starts_from: [],
		status,
		action: '',
		role_sha256: '',
		result,
		note: '',
		missing_fixtures: [],
		fcaf_replays: []
	};
}

describe('modePlatforms', () => {
	it('keeps every platform for a chat session', () => {
		expect(modePlatforms(new Set(['android', 'ios']), 'chat')).toEqual(
			new Set(['android', 'ios'])
		);
	});

	it('keeps android only for an FCAF session', () => {
		expect(modePlatforms(new Set(['android', 'ios']), 'fcaf')).toEqual(new Set(['android']));
	});

	it('has no platform for an FCAF session of a wallet without android builds', () => {
		expect(modePlatforms(new Set(['ios']), 'fcaf').size).toBe(0);
	});
});

describe('progressDone', () => {
	it('counts the roles neither pending nor being written', () => {
		const roles = [
			progress('a', 'passed'),
			progress('b', 'working'),
			progress('c', 'pending'),
			progress('d', 'blocked')
		];
		expect(progressDone(roles)).toBe(2);
	});
});

describe('roleStatusVariant', () => {
	it('marks a blocked role as destructive and a passed one as default', () => {
		expect(roleStatusVariant('blocked')).toBe('destructive');
		expect(roleStatusVariant('passed')).toBe('default');
	});

	it('marks a wallet finding as destructive', () => {
		expect(roleStatusVariant('finding')).toBe('destructive');
	});

	it('falls back to outline for an unknown status', () => {
		expect(roleStatusVariant('done')).toBe('outline');
	});
});

describe('roleStatusLabel', () => {
	it('shows an unknown status as is', () => {
		expect(roleStatusLabel('done')).toBe('done');
	});

	it('labels a wallet finding', () => {
		expect(roleStatusLabel('finding')).toBe('Wallet finding');
	});
});

describe('replayStatusLabel and replayStatusVariant', () => {
	it('labels and colours every replay status', () => {
		expect(replayStatusLabel('passed')).toBe('Passed');
		expect(replayStatusLabel('finding')).toBe('Wallet finding');
		expect(replayStatusLabel('action_failed')).toBe('Action failed');
		expect(replayStatusVariant('passed')).toBe('default');
		expect(replayStatusVariant('finding')).toBe('destructive');
		expect(replayStatusVariant('action_failed')).toBe('outline');
	});

	it('keeps an unknown status as is', () => {
		expect(replayStatusLabel('blocked')).toBe('blocked');
		expect(replayStatusVariant('blocked')).toBe('outline');
	});
});

describe('coverageVerdict', () => {
	it('shows the verdict of a current action', () => {
		expect(coverageVerdict(coverage('a', 'current', 'unsupported'))).toBe('unsupported');
		expect(coverageVerdict(coverage('a', 'current', 'passed'))).toBe('passed');
		expect(coverageVerdict(coverage('a', 'current', 'finding'))).toBe('finding');
	});

	it('labels a finding coverage as a destructive wallet finding', () => {
		expect(coverageLabel('finding')).toBe('Wallet finding');
		expect(coverageVariant('finding')).toBe('destructive');
	});

	it('marks a current action without a verdict as unrecorded', () => {
		expect(coverageVerdict(coverage('a', 'current'))).toBe('unrecorded');
	});

	it('keeps the status of a stale, missing or unmanaged role', () => {
		expect(coverageVerdict(coverage('a', 'stale', 'passed'))).toBe('stale');
		expect(coverageVerdict(coverage('a', 'missing'))).toBe('missing');
		expect(coverageVerdict(coverage('a', 'unmanaged'))).toBe('unmanaged');
	});

	it('shows an incomplete action as not replayed on every request', () => {
		const role = coverage('a', 'incomplete', 'passed');
		expect(coverageVerdict(role)).toBe('incomplete');
		expect(coverageLabel('incomplete')).toBe('Not replayed on every request');
		expect(coverageVariant('incomplete')).toBe('outline');
	});
});

describe('coverageCounts', () => {
	it('counts the written, the recorded and the supported roles', () => {
		const roles = [
			coverage('a', 'current', 'passed'),
			coverage('b', 'current', 'unsupported'),
			coverage('c', 'current'),
			coverage('d', 'missing')
		];
		expect(coverageCounts(roles)).toEqual({
			total: 4,
			written: 3,
			incomplete: 0,
			recorded: 2,
			supported: 1
		});
	});

	it('counts a wallet finding as recorded, not supported', () => {
		const roles = [coverage('a', 'current', 'passed'), coverage('b', 'current', 'finding')];
		expect(coverageCounts(roles)).toEqual({
			total: 2,
			written: 2,
			incomplete: 0,
			recorded: 2,
			supported: 1
		});
	});

	it('counts an incomplete action as written, not recorded', () => {
		const roles = [coverage('a', 'current', 'passed'), coverage('b', 'incomplete', 'passed')];
		expect(coverageCounts(roles)).toEqual({
			total: 2,
			written: 2,
			incomplete: 1,
			recorded: 1,
			supported: 1
		});
	});
});

describe('percent', () => {
	it('rounds the percentage', () => {
		expect(percent(2, 3)).toBe(67);
	});

	it('is 0 without roles', () => {
		expect(percent(0, 0)).toBe(0);
	});
});
