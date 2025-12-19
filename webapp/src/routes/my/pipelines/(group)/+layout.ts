// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { checkAuthFlagAndUser } from '$lib/utils';

//

export const load = async ({ fetch }) => {
	await checkAuthFlagAndUser({ fetch });
};
