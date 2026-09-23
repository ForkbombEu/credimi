// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getUserOrganization } from '$lib/utils';

/** Hub-wide: org for ownership / check-page namespace. Catalog loads stay route-local
 * (suite table on `+page`, FS nest on conformance-checks detail). */
export const load = async ({ fetch }) => {
	const organization = await getUserOrganization({ fetch });
	return { organization };
};
