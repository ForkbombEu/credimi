// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify';

import { getPipelineWorkflows } from '../_partials/workflows.js';

//

export const load = async ({ params, fetch }) => {
	const pipeline = await getRecordByCanonifiedPath(params.pipeline_id, { fetch });
	
	if (pipeline instanceof Error) {
		throw pipeline;
	}
	
	const workflows = await getPipelineWorkflows(pipeline.id, { fetch });

	return { pipeline, workflows };
};
