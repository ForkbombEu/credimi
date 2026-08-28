// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { TypedConfig } from '$pipeline-form/steps/types';

import { entities } from '$lib/global/entities.js';
import { stringify } from 'yaml';

import CardDetails from './card-details.svelte';
import {
	FCAFValidationStepForm,
	type FCAFValidationFormData,
	getFCAFValidationTestIDs,
	parseFCAFValidationConfiguration
} from './fcaf-validation-step-form.svelte.js';

//

export const fcafValidationStepConfig: TypedConfig<'fcaf-validation', FCAFValidationFormData> = {
	use: 'fcaf-validation',

	display: {
		...entities.conformance_checks,
		slug: 'fcaf-validation',
		labels: {
			singular: 'FCAF validation',
			plural: 'FCAF validations'
		}
	},

	initForm: (opts) => new FCAFValidationStepForm(opts),
	CardDetailsComponent: CardDetails,

	serialize: ({ yaml }) => parseFCAFValidationConfiguration(yaml),

	deserialize: async (data) => ({
		yaml: stringify(data)
	}),

	cardData: ({ yaml }) => {
		const data = parseFCAFValidationConfiguration(yaml);
		const testCount = getFCAFValidationTestIDs(yaml).length;
		return {
			title: 'FCAF validation',
			copyText: data.suite,
			meta: testCount > 0 ? { Tests: testCount } : undefined
		};
	},

	makeId: () => 'fcaf-validation'
};
