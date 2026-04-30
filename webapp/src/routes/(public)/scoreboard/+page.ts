// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Scoreboard } from '$lib';

export const load = async ({ fetch }) => {
	const data = await Scoreboard.loadData({ fetch });
	return {
		scoreboardData: data.items
	};
};
