// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListResult } from 'pocketbase';

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
	fetch?: typeof fetch;
};

export async function loadScoreboardData(
	options: Options = {}
): Promise<ListResult<ScoreboardRow>> {
	const res = await agent.getList(options.pagination?.page ?? 1, options.pagination?.perPage, {
		fetch: options.fetch
	});
	return res as ListResult<ScoreboardRow>;
}
