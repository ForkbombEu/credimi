// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

import type { OrgRole } from '.';

/* Reference: pb_hooks/organizations.routes.pb.js */

export async function verifyUserMembership(
	organizationId: string,
	fetchFn = fetch
): Promise<{ isMember: boolean }> {
	return await pb.send('/organizations/verify-user-membership', {
		method: 'POST',
		body: {
			organizationId
		},
		requestKey: null,
		fetch: fetchFn
	});
}

export async function verifyUserRole(
	organizationId: string,
	roles: OrgRole[],
	fetchFn = fetch
): Promise<{ hasRole: boolean }> {
	return await pb.send('/organizations/verify-user-role', {
		method: 'POST',
		body: {
			organizationId,
			roles
		},
		requestKey: null,
		fetch: fetchFn
	});
}
