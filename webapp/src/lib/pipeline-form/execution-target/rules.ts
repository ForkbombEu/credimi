// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { FormIntent } from '../steps/types';
import type { EnrichedStep } from '../steps-builder/types';

export function countMobileSteps(steps: EnrichedStep[]): number {
	return steps.filter(([raw]) => raw.use === 'mobile-automation').length;
}

export function shouldLockFormFields(opts: {
	intent: FormIntent;
	steps: EnrichedStep[];
}): boolean {
	const count = countMobileSteps(opts.steps);
	return opts.intent === 'add' ? count >= 1 : count >= 2;
}
