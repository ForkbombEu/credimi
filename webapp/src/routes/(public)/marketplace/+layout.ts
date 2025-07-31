// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getUserOrganization } from '$lib/utils';

export const load = async ({ fetch }) => {
	const organization = await getUserOrganization({ fetch });
	// Loading organization for displaying ownership status

	return {
		organization
	};
};
