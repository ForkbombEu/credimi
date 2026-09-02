// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListResult } from 'pocketbase';

import { ClientResponseError } from 'pocketbase';

import type { PipelineScoreboardCacheResponse } from '@/pocketbase/types';

import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent } from '@/pocketbase/query';

import type { ScoreboardRow } from '../types';

//

/** Public scoreboard listings must only include published pipelines.
 * Unpublished pipelines still exist in the cache, but their `pipeline`
 * expand is omitted for anonymous viewers, which left cards with no title. */
export const PUBLISHED_PIPELINE_FILTER = 'pipeline.published = true';

export function hasVisiblePipeline(row: ScoreboardRow): boolean {
	return Boolean(row.expand?.pipeline);
}

/** Always require published pipelines; optionally AND extra UI filters (e.g. score bands). */
export function buildLoadPageFilter(extraFilter?: string): string {
	return [PUBLISHED_PIPELINE_FILTER, extraFilter].filter(Boolean).join(' && ');
}

const agent = new PocketbaseQueryAgent({
	collection: 'pipeline_scoreboard_cache',
	expand: [
		'credentials',
		'custom_integrations',
		'issuers',
		'latest_execution',
		'mobile_devices',
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
	filter?: string;
	fetch?: typeof fetch;
};

type LoadForPipelineOptions = {
	fetch?: typeof fetch;
};

type LoadExecutionStatsForPipelineOptions = {
	fetch?: typeof fetch;
};

/** Unexpanded cache row — execution stats fields only (no relation expands). */
export type PipelineScoreboardCacheStats = PipelineScoreboardCacheResponse;

export async function loadPage(options: LoadPageOptions = {}): Promise<ListResult<ScoreboardRow>> {
	const res = await agent.getList(options.page ?? 1, options.perPage, {
		fetch: options.fetch,
		requestKey: null,
		filter: buildLoadPageFilter(options.filter),
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

export async function loadExecutionStatsForPipeline(
	pipelineId: string,
	options: LoadExecutionStatsForPipelineOptions = {}
): Promise<PipelineScoreboardCacheStats | undefined> {
	try {
		return await pb
			.collection('pipeline_scoreboard_cache')
			.getFirstListItem(pb.filter('pipeline = {:pipeline}', { pipeline: pipelineId }), {
				fetch: options.fetch,
				requestKey: null
			});
	} catch (error) {
		if (error instanceof ClientResponseError && (error.status === 404 || error.status === 0)) {
			return undefined;
		}
		console.error(error);
		return undefined;
	}
}
