// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Simplify } from 'type-fest';

import type {
	PipelineScoreboardCacheExpand,
	PipelineScoreboardCacheResponse
} from '@/pocketbase/types';

//

export type ScoreboardRow = Simplify<
	PipelineScoreboardCacheResponse<string[], PipelineScoreboardCacheExpand>
>;
