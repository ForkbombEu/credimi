// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Simplify } from 'type-fest';

import type { PipelineExecutionArtifacts } from '$lib/pipeline/execution-artifacts';

import type { PipelineScoreboardCacheResponse } from '@/pocketbase/types';

//

export type ScoreboardExpandedEntity = {
	id: string;
	collectionName: string;
	name?: string;
	logo?: string;
	logo_url?: string;
	published: boolean;
	__canonified_path__: string;
	wallet?: string;
	credential_issuer?: string;
	verifier?: string;
	tag?: string;
};

export type ScoreboardExpandedData = {
	pipeline?: ScoreboardExpandedEntity;
	mobile_runners: Array<{ id: string; name: string; description?: string }>;
	wallets: ScoreboardExpandedEntity[];
	wallet_versions: ScoreboardExpandedEntity[];
	issuers: ScoreboardExpandedEntity[];
	verifiers: ScoreboardExpandedEntity[];
	credentials: ScoreboardExpandedEntity[];
	use_case_verifications: ScoreboardExpandedEntity[];
	custom_integrations: ScoreboardExpandedEntity[];
	latest_successful_execution?: {
		created: string;
		artifacts: PipelineExecutionArtifacts;
	};
};

export type ScoreboardRow = Simplify<
	Omit<PipelineScoreboardCacheResponse<string[]>, 'expanded_data'> & {
		expanded_data?: ScoreboardExpandedData;
	}
>;
