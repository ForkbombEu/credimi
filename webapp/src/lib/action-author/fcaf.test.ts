// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type {
	AuthorDevice,
	FcafCoverageStatus,
	FcafPipelineRun,
	FcafRoleCoverage,
	FcafRoleProgress,
	FcafRoleRun,
	FcafRoleStatus,
	FcafRunSummary
} from './api';

import {
	coverageCounts,
	coverageLabel,
	coverageVariant,
	coverageVerdict,
	fcafDeviceRecords,
	fcafPreferredDevice,
	modePlatforms,
	percent,
	pipelineRunLink,
	progressDone,
	queueStatusLabel,
	replayStatusLabel,
	replayStatusVariant,
	roleRunBadge,
	roleRunFailing,
	roleStatusLabel,
	roleStatusVariant,
	runSummaryLine
} from './fcaf';

function progress(role: string, status: FcafRoleStatus): FcafRoleProgress {
	return { role, status, key: '', note: '' };
}

function coverage(
	role: string,
	status: FcafCoverageStatus,
	result: FcafRoleCoverage['result'] = '',
	run: FcafRoleRun | null = null
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
		fcaf_replays: [],
		run
	};
}

function roleRun(status: FcafRoleRun['status']): FcafRoleRun {
	return { status, failed_steps: 0, failed_command: '', fixture_id: '', run_url: '' };
}

function device(
	path: string,
	platform: string,
	opts: Partial<Pick<AuthorDevice, 'is_online' | 'agent_available' | 'runs_ahead'>> = {}
): AuthorDevice {
	return {
		path,
		name: path,
		runner_name: 'runner',
		runner_owner: 'org',
		model: 'model',
		type: 'android_phone',
		platform,
		is_online: true,
		agent_available: true,
		...opts
	};
}

describe('roleRunFailing and roleRunBadge', () => {
	it('marks a role failing in the last run with a destructive badge', () => {
		const role = coverage('a', 'current', 'passed', roleRun('failing'));
		expect(roleRunFailing(role)).toBe(true);
		expect(roleRunBadge(role.run)).toEqual({
			label: 'Failing in the last run',
			variant: 'destructive'
		});
	});

	it('keeps the coverage verdict of a role failing in the last run', () => {
		const role = coverage('a', 'current', 'passed', roleRun('failing'));
		expect(coverageVerdict(role)).toBe('passed');
	});

	it('labels a role that ran or was not run in the last run', () => {
		expect(roleRunBadge(roleRun('ok'))).toEqual({
			label: 'Ran in the last run',
			variant: 'secondary'
		});
		expect(roleRunBadge(roleRun('not_run'))).toEqual({
			label: 'Not run in the last run',
			variant: 'outline'
		});
		expect(roleRunFailing(coverage('a', 'current', 'passed', roleRun('ok')))).toBe(false);
	});

	it('shows no badge for a role the last run did not include', () => {
		expect(roleRunBadge(null)).toBeUndefined();
		expect(roleRunFailing(coverage('a', 'current'))).toBe(false);
	});
});

describe('runSummaryLine', () => {
	it('shows the passed tests over the total', () => {
		const summary: FcafRunSummary = {
			pass: 10,
			fail: 2,
			blocked: 1,
			skipped: 0,
			inconclusive: 0,
			not_applicable: 0,
			error: 0,
			total: 13,
			pipeline: 'org/fcaf-relying-party-validation-w',
			created: '2026-11-25 10:00:00.000Z',
			status: ''
		};
		expect(runSummaryLine(summary)).toBe('10 over 13 tests passed');
	});

	it('shows how a run without a report ended', () => {
		const summary: FcafRunSummary = {
			pass: 0,
			fail: 0,
			blocked: 0,
			skipped: 0,
			inconclusive: 0,
			not_applicable: 0,
			error: 0,
			total: 0,
			pipeline: 'org/fcaf-relying-party-validation-w',
			created: '2026-11-25 10:00:00.000Z',
			status: 'timed_out'
		};
		expect(runSummaryLine(summary)).toBe('Timed out before validating any test');
		expect(runSummaryLine({ ...summary, status: 'failed' })).toBe(
			'Failed before validating any test'
		);
	});
});

describe('pipelineRunLink', () => {
	const run: FcafPipelineRun = {
		pipeline: 'org/p',
		pipeline_url: 'https://example.org/my/pipelines/org/p',
		ticket_id: 't',
		device_id: 'org/runner/device',
		status: 'queued',
		skipped: [],
		test_count: 3
	};

	it('links the pipeline while the run has not started', () => {
		expect(pipelineRunLink(run)).toBe(run.pipeline_url);
	});

	it('links the run once it started', () => {
		const url = 'https://example.org/my/tests/runs/wf/run';
		expect(pipelineRunLink({ ...run, run_url: url })).toBe(url);
	});
});

describe('queueStatusLabel', () => {
	it('labels the queue statuses and keeps an unknown one', () => {
		expect(queueStatusLabel('queued')).toBe('In queue');
		expect(queueStatusLabel('running')).toBe('Running');
		expect(queueStatusLabel('not_found')).toBe('Not found');
		expect(queueStatusLabel('paused')).toBe('paused');
	});
});

describe('fcafDeviceRecords', () => {
	it('keeps the Android devices, whether their agent runs or not', () => {
		const records = fcafDeviceRecords([
			device('a', 'android', { agent_available: false }),
			device('i', 'ios'),
			device('b', 'android', { is_online: false })
		]);
		expect(records.map((r) => r.path)).toEqual(['a', 'b']);
		expect(records[0].unavailableReason).toBeUndefined();
		expect(records[1].isOnline).toBe(false);
	});
});

describe('fcafPreferredDevice', () => {
	it('prefers an online Android device with the fewest runs ahead, agent or not', () => {
		const preferred = fcafPreferredDevice([
			device('i', 'ios', { runs_ahead: 0 }),
			device('off', 'android', { is_online: false, runs_ahead: 0 }),
			device('busy', 'android', { runs_ahead: 2 }),
			device('free', 'android', { agent_available: false, runs_ahead: 0 })
		]);
		expect(preferred?.path).toBe('free');
	});

	it('finds none without an online Android device', () => {
		expect(fcafPreferredDevice([device('i', 'ios')])).toBeUndefined();
	});
});

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
