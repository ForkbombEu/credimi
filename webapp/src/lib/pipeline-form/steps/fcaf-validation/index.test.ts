// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { fcafValidationStepConfig } from '.';
import { getConfigByType, getDisplayData } from '..';

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
