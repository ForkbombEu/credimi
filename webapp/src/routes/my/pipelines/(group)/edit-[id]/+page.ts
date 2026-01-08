// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchPipeline } from '$lib/pipeline-form/functions';

import { redirect } from '@/i18n/index.js';

//

export const load = async ({ fetch, params }) => {
	try {
		// Try to fetch a "blocks" pipeline
		const pipeline = await fetchPipeline(params.id, { fetch });
		return {
			pipeline
		};
	} catch {
		// If the pipeline is not found, redirect to the manual edit page
		redirect(`/my/pipelines/edit-${params.id}/manual`);
	}
};
