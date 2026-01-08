// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { AnyPipelineStepConfig } from '../types';

import { conformanceCheckStepConfig } from './conformance-check';
import * as marketplaceSteps from './marketplace-item';
import { walletActionStepConfig } from './wallet-action';

//

export const configs: AnyPipelineStepConfig[] = [
	walletActionStepConfig,
	marketplaceSteps.credentialsStepConfig,
	marketplaceSteps.useCaseVerificationStepConfig,
	conformanceCheckStepConfig,
	marketplaceSteps.customCheckStepConfig
];
