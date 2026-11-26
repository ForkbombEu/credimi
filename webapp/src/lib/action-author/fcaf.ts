// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { DeviceRecord } from '$lib/pipeline/device/types';
import type { Status as QueueStatus } from '$lib/pipeline/queue';

import type { BadgeVariant } from '@/components/ui/badge';

import { m } from '@/i18n';
import { createSessionStorageHandlers } from '@/utils/storage';

import type {
	AuthorDevice,
	AuthoringMode,
	FcafPipelineRun,
	FcafPipelineVerdict,
	FcafReplay,
	FcafRoleCoverage,
	FcafRoleProgress,
	FcafRoleRun,
	FcafRunStatus,
	FcafRunSummary
} from './api';

import { toDeviceRecords } from './builds';

//

/** The build platforms a session of the mode can use: FCAF sessions run on Android only. */
export function modePlatforms(platforms: Set<string>, mode: AuthoringMode): Set<string> {
	if (mode === 'chat') return platforms;
	return platforms.has('android') ? new Set(['android']) : new Set();
}

/** The roles the campaign is done with: neither pending nor being written. */
export function progressDone(roles: FcafRoleProgress[]): number {
	return roles.filter((r) => r.status !== 'pending' && r.status !== 'working').length;
}

const STATUS_VARIANTS: Record<string, BadgeVariant> = {
	passed: 'default',
	pending: 'secondary',
	working: 'secondary',
	unsupported: 'outline',
	finding: 'destructive',
	skipped: 'outline',
	blocked: 'destructive'
};

/** The badge variant of a role's campaign status; outline for an unknown one. */
export function roleStatusVariant(status: string): BadgeVariant {
	return Object.hasOwn(STATUS_VARIANTS, status) ? STATUS_VARIANTS[status] : 'outline';
}

const STATUS_LABELS: Record<string, () => string> = {
	pending: m.Fcaf_role_pending,
	working: m.Fcaf_role_working,
	passed: m.Fcaf_role_passed,
	unsupported: m.Fcaf_role_unsupported,
	finding: m.Fcaf_role_finding,
	blocked: m.Fcaf_role_blocked,
	skipped: m.Fcaf_role_skipped
};

/** The label of a role's campaign status; the status itself for an unknown one. */
export function roleStatusLabel(status: string): string {
	return Object.hasOwn(STATUS_LABELS, status) ? STATUS_LABELS[status]() : status;
}

const REPLAY_LABELS: Record<FcafReplay['status'], () => string> = {
	passed: m.Fcaf_role_passed,
	finding: m.Fcaf_role_finding,
	action_failed: m.Fcaf_replay_action_failed
};

const REPLAY_VARIANTS: Record<FcafReplay['status'], BadgeVariant> = {
	passed: 'default',
	finding: 'destructive',
	action_failed: 'outline'
};

/** The label of a replay's status; the status itself for an unknown one. */
export function replayStatusLabel(status: string): string {
	return Object.hasOwn(REPLAY_LABELS, status)
		? REPLAY_LABELS[status as FcafReplay['status']]()
		: status;
}

/** The badge variant of a replay's status; outline for an unknown one. */
export function replayStatusVariant(status: string): BadgeVariant {
	return Object.hasOwn(REPLAY_VARIANTS, status)
		? REPLAY_VARIANTS[status as FcafReplay['status']]
		: 'outline';
}

/** What the wallet's coverage says of a role. */
export type CoverageVerdict =
	| 'passed'
	| 'unsupported'
	| 'finding'
	| 'failing'
	| 'blocked'
	| 'unrecorded'
	| 'incomplete'
	| 'stale'
	| 'missing'
	| 'unmanaged';

/**
 * The coverage verdict of a role: a current or incomplete action the last pipeline run covered
 * shows the pipeline's verdict; otherwise a current action shows the agent's verdict on it.
 */
export function coverageVerdict(role: FcafRoleCoverage): CoverageVerdict {
	switch (role.status) {
		case 'current':
		case 'incomplete':
			if (verifiedByPipeline(role)) return role.verdict as FcafPipelineVerdict;
			return role.status === 'current' ? role.result || 'unrecorded' : 'incomplete';
		case 'stale':
			return 'stale';
		case 'unmanaged':
			return 'unmanaged';
		default:
			return 'missing';
	}
}

/** Whether the role's verdict comes from the last pipeline run. */
export function verifiedByPipeline(role: FcafRoleCoverage): boolean {
	return role.verdict_source === 'pipeline' && role.verdict !== '';
}

/**
 * The roles with a current or incomplete action (written), those with an incomplete one
 * (incomplete: written, not replayed on every request), those whose current action has a
 * recorded verdict (recorded), those whose recorded verdict passed (supported), those the
 * pipeline verified (verified) and how many of those got each pipeline verdict (verdicts).
 */
export function coverageCounts(roles: FcafRoleCoverage[]): {
	total: number;
	written: number;
	incomplete: number;
	recorded: number;
	supported: number;
	verified: number;
	verdicts: Record<FcafPipelineVerdict, number>;
} {
	const current = roles.filter((r) => r.status === 'current');
	const incomplete = roles.filter((r) => r.status === 'incomplete').length;
	const recorded = current.filter((r) => r.result !== '');
	const verified = roles.filter(
		(r) => (r.status === 'current' || r.status === 'incomplete') && verifiedByPipeline(r)
	);
	const verdicts: Record<FcafPipelineVerdict, number> = {
		passed: 0,
		finding: 0,
		failing: 0,
		blocked: 0
	};
	for (const role of verified) {
		const verdict = role.verdict as FcafPipelineVerdict;
		if (Object.hasOwn(verdicts, verdict)) verdicts[verdict]++;
	}
	return {
		total: roles.length,
		written: current.length + incomplete,
		incomplete,
		recorded: recorded.length,
		supported: recorded.filter((r) => r.result === 'passed').length,
		verified: verified.length,
		verdicts
	};
}

/** n over total as a rounded percentage; 0 when total is 0. */
export function percent(n: number, total: number): number {
	return total === 0 ? 0 : Math.round((100 * n) / total);
}

const COVERAGE_LABELS: Record<CoverageVerdict, () => string> = {
	passed: m.Fcaf_role_passed,
	unsupported: m.Fcaf_role_unsupported,
	finding: m.Fcaf_role_finding,
	failing: m.Fcaf_run_role_failing,
	blocked: m.Fcaf_role_blocked,
	unrecorded: m.Fcaf_coverage_unrecorded,
	incomplete: m.Fcaf_coverage_incomplete,
	stale: m.Fcaf_coverage_stale,
	missing: m.Fcaf_coverage_missing,
	unmanaged: m.Fcaf_coverage_unmanaged
};

const COVERAGE_VARIANTS: Record<CoverageVerdict, BadgeVariant> = {
	passed: 'default',
	unsupported: 'outline',
	finding: 'destructive',
	failing: 'destructive',
	blocked: 'outline',
	unrecorded: 'secondary',
	incomplete: 'outline',
	stale: 'outline',
	missing: 'secondary',
	unmanaged: 'outline'
};

/** The label of a coverage verdict. */
export function coverageLabel(verdict: CoverageVerdict): string {
	return COVERAGE_LABELS[verdict]();
}

/** The badge variant of a coverage verdict. */
export function coverageVariant(verdict: CoverageVerdict): BadgeVariant {
	return COVERAGE_VARIANTS[verdict];
}

/** Whether the role's action failed in the wallet's last FCAF run. */
export function roleRunFailing(role: FcafRoleCoverage): boolean {
	return role.run?.status === 'failing';
}

/**
 * The roles a fix session works on: a current or incomplete action the wallet's last FCAF run
 * found failing.
 */
export function fixableRoles(roles: FcafRoleCoverage[]): FcafRoleCoverage[] {
	return roles.filter(
		(r) =>
			(r.status === 'current' || r.status === 'incomplete') &&
			r.verdict === 'failing' &&
			r.verdict_source === 'pipeline'
	);
}

/**
 * The scenarios to run again after a fix: those of the fixtures where the last FCAF run failed
 * roles whose current or incomplete action was saved after that run (run failing, verdict not
 * from the pipeline), except roles saved with a wallet finding (result finding: the kept action
 * fails there again by construction). Unique and sorted.
 */
export function affectedScenarios(roles: FcafRoleCoverage[]): string[] {
	const scenarios = new Set<string>();
	for (const role of roles) {
		if (role.status !== 'current' && role.status !== 'incomplete') continue;
		if (role.run?.verdict !== 'failing' || role.verdict_source === 'pipeline') continue;
		if (role.result === 'finding') continue;
		for (const fixture of role.run.failed_fixtures ?? []) {
			const scenario = fixture.split('#')[0];
			if (scenario) scenarios.add(scenario);
		}
	}
	return [...scenarios].sort();
}

const RUN_LABELS: Record<FcafRoleRun['status'], () => string> = {
	ok: m.Fcaf_run_role_ok,
	failing: m.Fcaf_run_role_failing,
	not_run: m.Fcaf_run_role_not_run
};

const RUN_VARIANTS: Record<FcafRoleRun['status'], BadgeVariant> = {
	ok: 'secondary',
	failing: 'destructive',
	not_run: 'outline'
};

/**
 * The badge of how a role's action did in the last FCAF run; none when that run did not include
 * it or when the role's verdict already comes from it.
 */
export function roleRunBadge(
	role: FcafRoleCoverage
): { label: string; variant: BadgeVariant } | undefined {
	const run = role.run;
	if (!run || verifiedByPipeline(role) || !Object.hasOwn(RUN_LABELS, run.status)) {
		return undefined;
	}
	return { label: RUN_LABELS[run.status](), variant: RUN_VARIANTS[run.status] };
}

const RUN_STATUS_LABELS: Record<FcafRunStatus, () => string> = {
	completed: m.Fcaf_run_status_completed,
	failed: m.Fcaf_run_status_failed,
	canceled: m.Fcaf_run_status_canceled,
	terminated: m.Fcaf_run_status_terminated,
	timed_out: m.Fcaf_run_status_timed_out,
	continued_as_new: m.Fcaf_run_status_continued_as_new
};

/** The label of a finished run's status; the status itself for an unknown one. */
function runStatusLabel(status: string): string {
	return Object.hasOwn(RUN_STATUS_LABELS, status)
		? RUN_STATUS_LABELS[status as FcafRunStatus]()
		: status;
}

const QUEUE_STATUS_LABELS: Record<QueueStatus, () => string> = {
	queued: m.Fcaf_queue_status_queued,
	starting: m.Fcaf_queue_status_starting,
	running: m.Fcaf_queue_status_running,
	failed: m.Fcaf_queue_status_failed,
	canceled: m.Fcaf_queue_status_canceled,
	not_found: m.Fcaf_queue_status_not_found
};

/** The label of a queued FCAF run's queue status; the status itself for an unknown one. */
export function queueStatusLabel(status: string): string {
	return Object.hasOwn(QUEUE_STATUS_LABELS, status)
		? QUEUE_STATUS_LABELS[status as QueueStatus]()
		: status;
}

/**
 * The last FCAF run's summary line: "N over M tests passed", or how the run ended when it
 * stopped before validating any test.
 */
export function runSummaryLine(summary: FcafRunSummary): string {
	if (summary.total === 0 && summary.status) {
		return m.Fcaf_run_ended_without_report({ status: runStatusLabel(summary.status) });
	}
	return m.Fcaf_run_summary({ pass: summary.pass, total: summary.total });
}

/** Where to follow a queued FCAF run: the run once it started, else its pipeline. */
export function pipelineRunLink(run: FcafPipelineRun): string {
	return run.run_url || run.pipeline_url;
}

/** The devices an FCAF validation can run on: Android ones; a pipeline needs no agent, only an online device. */
export function fcafDeviceRecords(devices: AuthorDevice[]): DeviceRecord[] {
	return toDeviceRecords(devices, new Set(['android'])).map((record) => ({
		...record,
		unavailableReason: undefined
	}));
}

/** The device an FCAF validation should use: an online Android one, fewest runs ahead first; ties keep list order. */
export function fcafPreferredDevice(devices: AuthorDevice[]): AuthorDevice | undefined {
	let preferred: AuthorDevice | undefined;
	for (const d of devices) {
		if (d.platform !== 'android' || !d.is_online) continue;
		if (!preferred || (d.runs_ahead ?? 0) < (preferred.runs_ahead ?? 0)) preferred = d;
	}
	return preferred;
}

/** The guidance typed in the start dialog, kept until the session page sends it as the first message. */
export const fcafGuidance = (sessionId: string) =>
	createSessionStorageHandlers<{ text: string }>(`action_author_fcaf_guidance:${sessionId}`);
