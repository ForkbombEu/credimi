// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { EnrichedStep } from '../steps-builder/types';

import { countMobileSteps, shouldLockFormFields } from './rules';

function mobileStep(): EnrichedStep {
	return [{ use: 'mobile-automation', id: 's1', continue_on_error: false, with: {} }, {}];
}

function debugStep(): EnrichedStep {
	return [{ use: 'debug' }, {}];
}

describe('countMobileSteps', () => {
	it('counts only mobile-automation steps', () => {
		expect(countMobileSteps([debugStep(), mobileStep(), mobileStep()])).toBe(2);
	});
});

describe('shouldLockFormFields', () => {
	it('add with 0 mobile steps → unlocked', () => {
		expect(shouldLockFormFields({ intent: 'add', steps: [] })).toBe(false);
	});

	it('add with 1+ mobile steps → locked', () => {
		expect(shouldLockFormFields({ intent: 'add', steps: [mobileStep()] })).toBe(true);
	});

	it('edit with 1 mobile step → unlocked', () => {
		expect(shouldLockFormFields({ intent: 'edit', steps: [mobileStep()] })).toBe(false);
	});

	it('edit with 2+ mobile steps → locked', () => {
		expect(
			shouldLockFormFields({ intent: 'edit', steps: [mobileStep(), mobileStep()] })
		).toBe(true);
	});
});
