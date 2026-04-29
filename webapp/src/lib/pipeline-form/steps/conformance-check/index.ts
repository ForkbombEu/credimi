// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepByType, PipelineStepData } from '$lib/pipeline/index.js';

import { entities } from '$lib/global/entities.js';
import { getStandardsWithTestSuites } from '$lib/standards';

import { localizeHref } from '@/i18n/index.js';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import { formatLinkedId } from '../utils.js';
import { ConformanceCheckStepForm, type FormData } from './conformance-check-step-form.svelte.js';

//

export const conformanceCheckStepConfig: TypedConfig<'conformance-check', FormData> = {
	use: 'conformance-check',

	display: entities.conformance_checks,

	initForm: () => new ConformanceCheckStepForm(),

	serialize: ({ test }) => {
		type StepData = PipelineStepData<PipelineStepByType<'conformance-check'>>;
		const _with: StepData = { check_id: test };
		if (test.startsWith('openid4vci_issuer')) {
			_with.credential_offer = '<credential-offer-placeholder>';
		}
		return _with;
	},

	linkProcedure: (serialized, previousSteps) => {
		if (!serialized.credential_offer) return;

		const previousStep = previousSteps
			.toReversed()
			.find((step) => step.use === 'credential-offer');
		if (!previousStep) return;

		serialized.credential_offer = formatLinkedId(previousStep);
	},

	makeId: ({ check_id }) => getLastPathSegment(check_id),

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
	},

	cardData: ({ suite, test, standard }) => {
		const testPath = suite.paths.find((path) => path.endsWith(test));
		return {
			title: test.split('/').at(-1)?.replaceAll('+', ' ') ?? '',
			copyText: test,
			avatar: suite.logo,
			meta: {
				standard: standard.name
			},
			publicUrl: localizeHref(`/marketplace/conformance-checks/${testPath}`)
		};
	}
};
