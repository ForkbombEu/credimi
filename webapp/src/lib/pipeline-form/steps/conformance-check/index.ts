// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepByType, PipelineStepData } from '$lib/pipeline/index.js';

import { entities } from '$lib/global/entities.js';
import { getStandardsWithTestSuites } from '$lib/standards';

import { localizeHref, m } from '@/i18n/index.js';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import { formatLinkedId } from '../utils.js';
import { ConformanceCheckStepForm, type FormData } from './conformance-check-step-form.svelte.js';

//

export const conformanceCheckStepConfig: TypedConfig<'conformance-check', FormData> = {
	use: 'conformance-check',

	display: entities.conformance_checks,

	initForm: (opts) => new ConformanceCheckStepForm(opts),

	serialize: ({ test, action_id }) => {
		type StepData = PipelineStepData<PipelineStepByType<'conformance-check'>>;
		const _with: StepData = { check_id: test };
		if (test.startsWith('openid4vci_issuer') || test.startsWith('openid4vp_verifier')) {
			_with.parameters = { deeplink: '<placeholder>' };
		} else if (test.startsWith('openid4vci_wallet')) {
			if (!action_id) {
				throw new Error(m.Pipeline_form_missing_wallet_action_openid4vci_wallet_check());
			}
			_with.parameters = {
				workflow_id: '${{ workflow_id }}',
				run_id: '${{ run_id }}',
				organization_id: '${{ organization_id }}',
				action_id
			};
		}
		return _with;
	},

	linkProcedure: (serialized, previousSteps) => {
		if (!serialized.parameters?.deeplink) return;

		const previousStep = previousSteps
			.toReversed()
			.find(
				(step) =>
					step.use === 'credential-offer' || step.use === 'use-case-verification-deeplink'
			);
		if (!previousStep) return;

		serialized.parameters.deeplink = formatLinkedId(previousStep);
		serialized.parameters.use_case_id = previousStep.with?.use_case_id;
		serialized.parameters.credential_id = previousStep.with?.credential_id;
	},

	makeId: ({ check_id }) => getLastPathSegment(check_id),

	deserialize: async ({ check_id, parameters }) => {
		const chunks = check_id.split('/');
		if (chunks.length !== 4) throw new Error(m.Pipeline_form_invalid_check_id());

		const [standardUid, versionUid, suiteUid, test] = chunks;
		const standardsWithTestSuites = await getStandardsWithTestSuites();
		if (standardsWithTestSuites instanceof Error) throw standardsWithTestSuites;

		const standard = standardsWithTestSuites.find((standard) => standard.uid === standardUid);
		const version = standard?.versions.find((version) => version.uid === versionUid);
		const suite = version?.suites.find((suite) => suite.uid === suiteUid);

		if (!standard || !version || !suite)
			throw new Error(m.Pipeline_form_standard_version_or_suite_not_found());

		return {
			standard,
			version,
			suite,
			test,
			action_id: typeof parameters?.action_id === 'string' ? parameters.action_id : undefined
		};
	},

	cardData: ({ suite, test, standard }) => {
		const testPath = suite.paths.find((path) => path.endsWith(test));
		if (testPath === undefined) {
			throw new Error(m.Pipeline_form_conformance_check_path_not_found());
		}
		return {
			title: test.split('/').at(-1) ?? '',
			copyText: test,
			avatar: suite.logo,
			meta: {
				standard: standard.name
			},
			publicUrl: localizeHref(`/hub/conformance-checks/${testPath}`)
		};
	}
};
