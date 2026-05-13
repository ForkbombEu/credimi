// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListResult } from 'pocketbase';

import { ClientResponseError } from 'pocketbase';

import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent } from '@/pocketbase/query';

import type { ScoreboardRow } from '../types';

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

type LoadPageOptions = {
	page?: number;
	perPage?: number;
	sort?: string;
	fetch?: typeof fetch;
};

type LoadForPipelineOptions = {
	fetch?: typeof fetch;
};

export async function loadPage(options: LoadPageOptions = {}): Promise<ListResult<ScoreboardRow>> {
	const res = await agent.getList(options.page ?? 1, options.perPage, {
		fetch: options.fetch,
		requestKey: null,
		...(options.sort ? { sort: options.sort } : {})
	});
	return res as ListResult<ScoreboardRow>;
}

export async function loadForPipeline(
	pipelineId: string,
	options: LoadForPipelineOptions = {}
): Promise<ScoreboardRow | undefined> {
	try {
		const res = await agent.getList(1, 1, {
			fetch: options.fetch,
			requestKey: null,
			filter: pb.filter('pipeline = {:pipeline}', { pipeline: pipelineId })
		});
		return res.items[0] as ScoreboardRow | undefined;
	} catch (error) {
		if (error instanceof ClientResponseError && (error.status === 404 || error.status === 0)) {
			return undefined;
		}
		console.error(error);
		return undefined;
	}
}
