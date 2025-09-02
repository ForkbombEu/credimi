// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Option as O } from 'effect';

import { pb } from '@/pocketbase';
import { type OrgAuthorizationsResponse, type OrgRolesResponse } from '@/pocketbase/types';

import type { OrgRole } from '.';

export async function getUserRole(organizationId: string, userId: string): Promise<OrgRole> {
	type AuthorizationWithRole = OrgAuthorizationsResponse<{ role: OrgRolesResponse }>;

	const orgAuthorization = await pb
		.collection('orgAuthorizations')
		.getFirstListItem<AuthorizationWithRole>(
			`organization.id = '${organizationId}' && user.id = '${userId}'`,
			{
				expand: 'role',
				requestKey: null
			}
		);

	return O.fromNullable(orgAuthorization.expand?.role.name).pipe(O.getOrThrow) as OrgRole;
}
