// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getConfigByType } from '$lib/pipeline-form/steps';

import type { GenericRecord } from '@/utils/types';

import { Enrich404Error, type EnrichedStep } from '../types';

//

export function getStepData(step: EnrichedStep): GenericRecord | undefined {
	if (step[0].use === 'debug') return undefined;
	if (step[1] instanceof Enrich404Error || step[1] instanceof Error) return undefined;
	return step[1];
}

export function isStepEditable(step: EnrichedStep): boolean {
	if (step[0].use === 'debug') return false;
	return getStepData(step) !== undefined && getConfigByType(step[0].use) !== undefined;
}
