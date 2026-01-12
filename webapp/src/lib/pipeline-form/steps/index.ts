// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { AnyConfig } from './types';

import { conformanceCheckStepConfig } from './conformance-check';
import * as marketplaceSteps from './marketplace-item';
import * as utilsSteps from './utils';
import { walletActionStepConfig } from './wallet-action';

//

export const configs: AnyConfig[] = [
	walletActionStepConfig,
	marketplaceSteps.credentialsStepConfig,
	marketplaceSteps.useCaseVerificationStepConfig,
	conformanceCheckStepConfig,
	marketplaceSteps.customCheckStepConfig,
	utilsSteps.emailStepConfig,
	utilsSteps.httpRequestStepConfig
];

export * from './types';
export * from './utils';
