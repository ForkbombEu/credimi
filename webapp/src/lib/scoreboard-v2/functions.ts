// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { PocketbaseQueryAgent } from '@/pocketbase/query';

import type { ScoreboardRow } from './types';

//

const agent = new PocketbaseQueryAgent({
	collection: 'pipeline_scoreboard_cache',
	expand: [
		'credentials',
		'custom_integrations',
		'issuers',
		'latest_successful_execution',
		'mobile_runners',
		'pipeline',
		'use_case_verifications',
		'verifiers',
		'wallet_versions',
		'wallets'
	]
});

type Options = {
	pagination?: {
		page: number;
		perPage: number;
	};
};

export function loadScoreboardData(options: Options = {}): Promise<ScoreboardRow[]> {
	return agent.getFullList({
		perPage: options.pagination?.perPage,
		page: options.pagination?.page ?? 1
	}) as Promise<ScoreboardRow[]>;
}
