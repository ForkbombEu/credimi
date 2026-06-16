// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getEnrichedPipeline } from '$lib/pipeline-form/functions';
import { getManualEditHref } from '$lib/pipeline/utils';

import { redirect } from '@/i18n/index.js';

//

export const load = async ({ fetch, parent }) => {
	const { pipeline } = await parent();

	if (pipeline.manual) {
		redirect(getManualEditHref(pipeline));
	}

	try {
		const enriched = await getEnrichedPipeline(pipeline.id, { fetch });
		return {
			pipeline: enriched
		};
	} catch {
		// If the pipeline is not found, redirect to the manual edit page
		redirect(getManualEditHref(pipeline));
	}
};
