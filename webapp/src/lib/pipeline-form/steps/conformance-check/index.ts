// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { TypedPipelineStepConfig } from '$lib/pipeline-form/types';

import { entities } from '$lib/global/entities.js';
import { getStandardsWithTestSuites } from '$lib/standards';

import { ConformanceCheckStepForm, type FormData } from './conformance-check-step-form.svelte.js';

//

export const conformanceCheckStepConfig: TypedPipelineStepConfig<'conformance-check', FormData> = {
	id: 'conformance-check',
	display: entities.conformance_checks,

	initForm: () => new ConformanceCheckStepForm(),

	serialize: ({ test }) => ({ check_id: test }),

	deserialize: async ({ check_id }) => {
		const chunks = check_id.split('/');
		if (chunks.length !== 4) throw new Error('Invalid check_id');

		const [standardUid, versionUid, suiteUid, test] = chunks;
		const standardsWithTestSuites = await getStandardsWithTestSuites();
		if (standardsWithTestSuites instanceof Error) throw standardsWithTestSuites;

		const standard = standardsWithTestSuites.find((standard) => standard.uid === standardUid);
		const version = standard?.versions.find((version) => version.uid === versionUid);
		const suite = version?.suites.find((suite) => suite.uid === suiteUid);

		if (!standard || !version || !suite)
			throw new Error('Standard, version, or suite not found');

		return {
			standard,
			version,
			suite,
			test
		};
	}
};
