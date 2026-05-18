// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Scoreboard } from '$lib';

export const load = async ({ fetch }) => {
	const data = await Scoreboard.Records.loadPage({
		fetch,
		perPage: 20,
		page: 1,
		sort: '-latest_successful_execution.created'
	});
	return {
		scoreboardData: data.items
	};
};
