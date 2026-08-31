// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_TESTS } from '$lib/fcaf/tests.generated.js';
import { createInitFormOptions } from '$pipeline-form/steps/init-form-options.test-utils.js';
import { describe, expect, it } from 'vitest';

import { fcafValidationStepConfig } from '.';
import { getConfigByType, getDisplayData } from '..';
import {
	FCAFValidationStepForm,
	parseFCAFValidationConfiguration
} from './fcaf-validation-step-form.svelte.js';

const configuration = {
	suite: 'wallet_solution/relying_party',
	test_ids: ['WS_RP_IA_Engagement__001', 'WS_RP_SM_DeviceBinding__007'],
	pipeline_outputs: {
		'example/scenario': {
			output: {
				evidence: '${{ scenario.outputs | optional }}'
			}
		}
	}
};

describe('FCAF validation step config', () => {
	it('is registered for pipeline editing', () => {
		expect(getConfigByType('fcaf-validation')).toBe(fcafValidationStepConfig);
		expect(getDisplayData('fcaf-validation').labels.singular).toBe('FCAF validation');
	});

	it('round-trips validation configuration', async () => {
		const formData = await fcafValidationStepConfig.deserialize(configuration);

		expect(fcafValidationStepConfig.serialize(formData)).toEqual(configuration);
	});

	it('shows suite and test count on card', async () => {
		const formData = await fcafValidationStepConfig.deserialize(configuration);

		expect(fcafValidationStepConfig.cardData(formData)).toEqual({
			title: 'FCAF validation',
			copyText: 'wallet_solution/relying_party',
			meta: { Tests: 2 }
		});
	});

	it('rejects non-object YAML', () => {
		expect(() => fcafValidationStepConfig.serialize({ yaml: 'invalid' })).toThrow(
			'FCAF validation configuration must be a YAML object'
		);
	});
});

describe('FCAF validation test selection', () => {
	it('toggles test ids through the yaml source of truth', () => {
		const form = new FCAFValidationStepForm(createInitFormOptions({ intent: 'add' }));

		expect(form.selectedTestIds).toEqual([]);

		form.toggleTestId('WS_RP_IA_Engagement__001');
		expect(form.selectedTestIds).toEqual(['WS_RP_IA_Engagement__001']);

		form.toggleTestId('WS_RP_SM_DeviceBinding__007');
		expect(form.selectedTestIds).toEqual([
			'WS_RP_IA_Engagement__001',
			'WS_RP_SM_DeviceBinding__007'
		]);

		form.toggleTestId('WS_RP_IA_Engagement__001');
		expect(form.selectedTestIds).toEqual(['WS_RP_SM_DeviceBinding__007']);
	});

	it('selects all available test ids and clears them', () => {
		const form = new FCAFValidationStepForm(createInitFormOptions({ intent: 'add' }));

		form.selectAllTestIds();
		expect(form.selectedTestIds).toHaveLength(form.availableTests.length);

		form.clearTestIds();
		expect(form.selectedTestIds).toEqual([]);
	});

	it('filters pipeline_outputs to only the selected test sources', () => {
		const form = new FCAFValidationStepForm(createInitFormOptions({ intent: 'add' }));
		const test = FCAF_TESTS[0];

		form.setTestIds([test.id]);

		const config = parseFCAFValidationConfiguration(form.data.yaml) as Record<string, unknown>;
		const outputs = config.pipeline_outputs as Record<string, unknown>;
		expect(Object.keys(outputs).sort()).toEqual([...test.sources].sort());
	});
});
