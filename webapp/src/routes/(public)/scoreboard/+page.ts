// SPDX-FileCopyrightText: 2025 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchAllResults } from '$lib/scoreboard';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const data = await fetchAllResults();
	return {
		scoreboardData: data
	};
};
