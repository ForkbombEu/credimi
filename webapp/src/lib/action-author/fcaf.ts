// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

import type {
	AuthoringMode,
	FcafCoverageStatus,
	FcafRoleCoverage,
	FcafRoleProgress,
	FcafRoleStatus
} from './api';

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

export type RoleStatusVariant = 'default' | 'secondary' | 'outline' | 'destructive';

const STATUS_VARIANTS: Record<FcafRoleStatus | FcafCoverageStatus, RoleStatusVariant> = {
	passed: 'default',
	current: 'default',
	pending: 'secondary',
	working: 'secondary',
	missing: 'secondary',
	unsupported: 'outline',
	skipped: 'outline',
	stale: 'outline',
	unmanaged: 'outline',
	blocked: 'destructive'
};

/** The badge variant of a role's campaign or coverage status. */
export function roleStatusVariant(status: FcafRoleStatus | FcafCoverageStatus): RoleStatusVariant {
	return STATUS_VARIANTS[status];
}

const STATUS_LABELS: Record<FcafRoleStatus | FcafCoverageStatus, () => string> = {
	pending: m.Fcaf_role_pending,
	working: m.Fcaf_role_working,
	passed: m.Fcaf_role_passed,
	unsupported: m.Fcaf_role_unsupported,
	blocked: m.Fcaf_role_blocked,
	skipped: m.Fcaf_role_skipped,
	missing: m.Fcaf_coverage_missing,
	current: m.Fcaf_coverage_current,
	stale: m.Fcaf_coverage_stale,
	unmanaged: m.Fcaf_coverage_unmanaged
};

/** The label of a role's campaign or coverage status. */
export function roleStatusLabel(status: FcafRoleStatus | FcafCoverageStatus): string {
	return STATUS_LABELS[status]();
}

/** The roles whose action is current. */
export function coverageCurrent(roles: FcafRoleCoverage[]): number {
	return roles.filter((r) => r.status === 'current').length;
}
