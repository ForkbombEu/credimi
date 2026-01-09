// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getEnrichedPipeline } from '$lib/pipeline-form/functions';

import { redirect } from '@/i18n/index.js';

//

export const load = async ({ fetch, params }) => {
	try {
		const pipeline = await getEnrichedPipeline(params.id, { fetch });
		return {
			pipeline
		};
	} catch {
		// If the pipeline is not found, redirect to the manual edit page
		redirect(`/my/pipelines/edit-${params.id}/manual`);
	}
};
