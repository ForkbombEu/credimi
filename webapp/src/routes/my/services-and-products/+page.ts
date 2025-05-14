// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getUserOrganization } from '$lib/utils/index.js';

export const load = async ({ fetch }) => {
	return await getUserOrganization({ fetch });
};
