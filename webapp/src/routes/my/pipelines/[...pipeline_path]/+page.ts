// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline, Scoreboard } from '$lib';
import { getRecordByCanonifiedPath } from '$lib/canonify';

import type { PipelinesResponse } from '@/pocketbase/types/index.generated.js';

import { getPaginationQueryParams, getStatusQueryParam } from '../../tests/runs/_partials';

//

export const load = async ({ params, fetch, url }) => {
	const pipeline = await getRecordByCanonifiedPath<PipelinesResponse>(params.pipeline_path, {
		fetch
	});
	if (pipeline instanceof Error) {
		throw pipeline;
	}

	const status = getStatusQueryParam(url);
	const pagination = getPaginationQueryParams(url);

	const [workflows, scoreboard] = await Promise.all([
		Pipeline.Workflows.list(pipeline.id, {
			fetch,
			status,
			...pagination
		}),
		Scoreboard.Records.loadForPipeline(pipeline.id, { fetch })
	]);

	return { pipeline, workflows, pagination, scoreboard };
};
