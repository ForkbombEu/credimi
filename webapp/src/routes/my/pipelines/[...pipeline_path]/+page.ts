// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline } from '$lib';
import { getRecordByCanonifiedPath } from '$lib/canonify';

import type { PipelinesResponse } from '@/pocketbase/types/index.generated.js';

//

export const load = async ({ params, fetch }) => {
	const pipeline = await getRecordByCanonifiedPath<PipelinesResponse>(params.pipeline_path, {
		fetch
	});
	if (pipeline instanceof Error) {
		throw pipeline;
	}

	const workflows = await Pipeline.Workflows.list(pipeline.id, { fetch });

	return { pipeline, workflows };
};
