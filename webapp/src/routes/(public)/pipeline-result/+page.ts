// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { validatePipelineReportUrl } from '$lib/pipeline/results/pipeline-report-page';

export const load = ({ url }) => {
	const mode = url.searchParams.get('mode');
	if (mode !== 'video' && mode !== 'screenshot') error(400, { message: 'Invalid mode' });

	const validated: Record<'video' | 'screenshot' | 'timeline', string | undefined> = {
		video: undefined,
		screenshot: undefined,
		timeline: undefined
	};
	for (const key of ['video', 'screenshot', 'timeline'] as const) {
		const raw = url.searchParams.get(key);
		if (!raw) continue;
		const valid = validatePipelineReportUrl(raw, url.origin);
		if (!valid) error(400, { message: `Invalid ${key} URL` });
		validated[key] = valid;
	}

	const { video, screenshot, timeline } = validated;
	if (!timeline) error(400, { message: 'Missing timeline URL' });
	if (mode === 'video' && !video) error(400, { message: 'Missing video URL' });
	if (mode === 'screenshot' && !screenshot) error(400, { message: 'Missing screenshot URL' });

	return { mode, video, screenshot, timeline };
};
