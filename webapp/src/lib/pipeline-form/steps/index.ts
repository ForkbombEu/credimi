// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { AnyConfig } from './types';

import { conformanceCheckStepConfig } from './conformance-check';
import * as marketplaceSteps from './marketplace-item';
import * as utilsSteps from './utils-steps';
import { walletActionStepConfig } from './wallet-action';

//

export const utilsConfigs: AnyConfig[] = [
	utilsSteps.emailStepConfig,
	utilsSteps.httpRequestStepConfig
];

export const coreConfigs: AnyConfig[] = [
	walletActionStepConfig,
	marketplaceSteps.credentialsStepConfig,
	marketplaceSteps.useCaseVerificationStepConfig,
	conformanceCheckStepConfig,
	marketplaceSteps.customCheckStepConfig
];

export const configs: AnyConfig[] = [...coreConfigs, ...utilsConfigs];

export * from './types';
export * from './utils';
