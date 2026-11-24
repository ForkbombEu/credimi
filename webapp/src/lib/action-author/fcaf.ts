// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { BadgeVariant } from '@/components/ui/badge';

import { m } from '@/i18n';
import { createSessionStorageHandlers } from '@/utils/storage';

import type { AuthoringMode, FcafRoleCoverage, FcafRoleProgress } from './api';

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
	blocked: m.Fcaf_role_blocked,
	skipped: m.Fcaf_role_skipped
};

/** The label of a role's campaign status; the status itself for an unknown one. */
export function roleStatusLabel(status: string): string {
	return Object.hasOwn(STATUS_LABELS, status) ? STATUS_LABELS[status]() : status;
}

/** What the wallet's coverage says of a role. */
export type CoverageVerdict =
	| 'passed'
	| 'unsupported'
	| 'unrecorded'
	| 'stale'
	| 'missing'
	| 'unmanaged';

/** The coverage verdict of a role: a current action shows the agent's verdict on it. */
export function coverageVerdict(role: FcafRoleCoverage): CoverageVerdict {
	switch (role.status) {
		case 'current':
			return role.result || 'unrecorded';
		case 'stale':
			return 'stale';
		case 'unmanaged':
			return 'unmanaged';
		default:
			return 'missing';
	}
}

/** The roles with a current action (written) and those whose current action passed (supported). */
export function coverageCounts(roles: FcafRoleCoverage[]): {
	total: number;
	written: number;
	supported: number;
} {
	const current = roles.filter((r) => r.status === 'current');
	return {
		total: roles.length,
		written: current.length,
		supported: current.filter((r) => r.result === 'passed').length
	};
}

/** n over total as a rounded percentage; 0 when total is 0. */
export function percent(n: number, total: number): number {
	return total === 0 ? 0 : Math.round((100 * n) / total);
}

const COVERAGE_LABELS: Record<CoverageVerdict, () => string> = {
	passed: m.Fcaf_role_passed,
	unsupported: m.Fcaf_role_unsupported,
	unrecorded: m.Fcaf_coverage_unrecorded,
	stale: m.Fcaf_coverage_stale,
	missing: m.Fcaf_coverage_missing,
	unmanaged: m.Fcaf_coverage_unmanaged
};

const COVERAGE_VARIANTS: Record<CoverageVerdict, BadgeVariant> = {
	passed: 'default',
	unsupported: 'outline',
	unrecorded: 'secondary',
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

/** The guidance typed in the start dialog, kept until the session page sends it as the first message. */
export const fcafGuidance = (sessionId: string) =>
	createSessionStorageHandlers<{ text: string }>(`action_author_fcaf_guidance:${sessionId}`);
