// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { validatePipelineReportUrl } from '$lib/pipeline/results/pipeline-report-page';

export const load = ({ url }) => {
	const rawReportUrl = url.searchParams.get('url');
	if (!rawReportUrl) error(400, { message: 'Missing report URL' });

	const reportUrl = validatePipelineReportUrl(rawReportUrl, url.origin);
	if (!reportUrl) error(400, { message: 'Invalid report URL' });

	return { reportUrl };
};
