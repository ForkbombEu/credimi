// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineExecutionArtifacts } from '$lib/pipeline/execution-artifacts';

import type { PipelineScoreboardCacheResponse } from '@/pocketbase/types';

//

export type ScoreboardExpandedEntity = {
	id: string;
	collectionName: string;
	name?: string;
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
	mobile_devices: Array<{
		id: string;
		device_id: string;
		name: string;
		runner_name: string;
		description?: string;
		type?: string;
	}>;
	wallets: ScoreboardExpandedEntity[];
	wallet_versions: ScoreboardExpandedEntity[];
	issuers: ScoreboardExpandedEntity[];
	verifiers: ScoreboardExpandedEntity[];
	credentials: ScoreboardExpandedEntity[];
	use_case_verifications: ScoreboardExpandedEntity[];
	custom_integrations: ScoreboardExpandedEntity[];
	latest_execution?: {
		created: string;
		artifacts: PipelineExecutionArtifacts;
	};
};

export type ScoreboardRow = PipelineScoreboardCacheResponse<string[], ScoreboardExpandedData>;
