// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { enrichPipeline } from '$lib/pipeline-form/functions';

import { redirect } from '@/i18n/index.js';
import { pb } from '@/pocketbase/index.js';

//

export const load = async ({ fetch, params }) => {
	try {
		const pipelineRecord = await pb.collection('pipelines').getOne(params.id, { fetch });
		const pipeline = await enrichPipeline(pipelineRecord);
		return {
			pipeline
		};
	} catch {
		// If the pipeline is not found, redirect to the manual edit page
		redirect(`/my/pipelines/edit-${params.id}/manual`);
	}
};
