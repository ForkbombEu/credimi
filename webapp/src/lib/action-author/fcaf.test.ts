// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { FcafCoverageStatus, FcafRoleCoverage, FcafRoleProgress, FcafRoleStatus } from './api';

import { coverageCurrent, modePlatforms, progressDone, roleStatusVariant } from './fcaf';

function progress(role: string, status: FcafRoleStatus): FcafRoleProgress {
	return { role, status, key: '', note: '' };
}

function coverage(role: string, status: FcafCoverageStatus): FcafRoleCoverage {
	return {
		role,
		summary: '',
		outcome: '',
		category: '',
		starts_from: [],
		status,
		action: '',
		role_sha256: ''
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
	it('marks a blocked role as destructive and a current one as default', () => {
		expect(roleStatusVariant('blocked')).toBe('destructive');
		expect(roleStatusVariant('current')).toBe('default');
	});
});

describe('coverageCurrent', () => {
	it('counts the roles whose action is current', () => {
		const roles = [
			coverage('a', 'current'),
			coverage('b', 'stale'),
			coverage('c', 'current'),
			coverage('d', 'missing')
		];
		expect(coverageCurrent(roles)).toBe(2);
	});
});
