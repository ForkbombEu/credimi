// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

import { getPipelineWorkflows } from '../_partials/workflows.js';

//

export const load = async ({ params, fetch }) => {
	const pipeline = await pb.collection('pipelines').getOne(params.pipeline_id, { fetch });
	const workflows = await getPipelineWorkflows(pipeline.id, { fetch });

	return { pipeline, workflows };
};
