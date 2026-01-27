// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify';

import { pb } from '@/pocketbase';

import { getPipelineWorkflows } from '../_partials/workflows.js';

//

export const load = async ({ params, fetch }) => {
	// Try to get pipeline by canonified path first, fallback to ID
	let pipeline;
	const pathOrId = params.pipeline_id;
	
	// Check if it looks like a canonified path (contains slash or dash)
	if (pathOrId.includes('/') || pathOrId.includes('-')) {
		const result = await getRecordByCanonifiedPath(pathOrId, { fetch });
		if (!(result instanceof Error)) {
			pipeline = result;
		}
	}
	
	// If not found by canonified path or doesn't look like a path, try by ID
	if (!pipeline) {
		pipeline = await pb.collection('pipelines').getOne(pathOrId, { fetch });
	}
	
	const workflows = await getPipelineWorkflows(pipeline.id, { fetch });

	return { pipeline, workflows };
};
