// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepType } from '$lib/pipeline/types';

import type { AnyConfig } from './types';

import { conformanceCheckStepConfig } from './conformance-check';
import * as hubSteps from './hub-item';
import * as utilsSteps from './utils-steps';
import { walletActionStepConfig } from './wallet-action';

//

export const utilsConfigs: AnyConfig[] = [
	utilsSteps.emailStepConfig,
	utilsSteps.httpRequestStepConfig
];

export const coreConfigs: AnyConfig[] = [
	walletActionStepConfig,
	hubSteps.credentialsStepConfig,
	hubSteps.useCaseVerificationStepConfig,
	conformanceCheckStepConfig,
	hubSteps.customCheckStepConfig
];

export const configs: AnyConfig[] = [...coreConfigs, ...utilsConfigs];

export function getConfigByType(type: PipelineStepType): AnyConfig {
	const config = configs.find((c) => c.use === type);
	if (!config) throw new Error(`Unknown step type: ${type}`);
	return config;
}

export * from './types';
export * from './utils';
