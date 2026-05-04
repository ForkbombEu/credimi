// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SetOptional, Simplify } from 'type-fest';

import type {
	PipelineScoreboardCacheExpand,
	PipelineScoreboardCacheResponse
} from '@/pocketbase/types';

//

export type ScoreboardRow = Simplify<
	PipelineScoreboardCacheResponse<
		string[],
		SetOptional<PipelineScoreboardCacheExpand, 'pipeline'>
		// Generated types say that the pipeline field is always present
		// but it's not always the case: pipelines can be "private"
		// so they exist in the relation but are not visible to all users
	>
>;
