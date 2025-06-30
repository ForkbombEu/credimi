// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { LatestCheckRunsStorage } from '$lib/start-checks-form/_utils';

export const load = async () => {
	if (!browser) return;

	const latestCheckRuns = LatestCheckRunsStorage.get();
	return {
		latestCheckRuns
	};
};
