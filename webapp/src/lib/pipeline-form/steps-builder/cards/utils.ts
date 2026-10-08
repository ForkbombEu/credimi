// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';

import { getConfigByType } from '$pipeline-form/steps';
import { isError } from 'effect/Predicate';

import type { GenericRecord } from '@/utils/types';

//

export function getStepError(step: EnrichedStep): Error | undefined {
	return isError(step[1]) ? step[1] : undefined;
}

export function getStepData(step: EnrichedStep): GenericRecord | undefined {
	if (step[0].use === 'debug') return undefined;
	if (getStepError(step)) return undefined;
	return step[1] as GenericRecord;
}

export function isStepEditable(step: EnrichedStep): boolean {
	if (step[0].use === 'debug') return false;
	return getStepData(step) !== undefined && getConfigByType(step[0].use) !== undefined;
}
