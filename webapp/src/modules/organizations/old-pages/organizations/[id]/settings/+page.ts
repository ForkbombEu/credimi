// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { blockMembersWithoutRoles } from '@/organizations';

export const load = async ({ params, fetch }) => {
	const organizationId = params.id;
	await blockMembersWithoutRoles(organizationId, ['owner'], fetch);
};
