// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { Array, pipe, String } from 'effect';

import type { HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition as Pipeline } from './pipeline.types.generated';

import {
	StepType,
	type BuilderStep,
	type CredentialStep,
	type CustomCheckStep,
	type UseCaseVerificationStep,
	type WalletActionStep
} from './steps-builder/types';

/*
 * TOC
 * -  Steps processing
 * -  YAML formatting
 */

/* Steps processing */

type YamlSteps = NonNullable<Pipeline['steps']>[number];
type YamlStepId = YamlSteps['use'];
type YamlStep<Id extends YamlStepId> = Extract<YamlSteps, { use: Id }>;
type AnyYamlStep = YamlStep<YamlStepId>;

export function convertBuilderSteps(steps: BuilderStep[]): AnyYamlStep[] {
	return pipe(steps, Array.map(convertStep), linkIds);
}

function convertStep(step: BuilderStep): AnyYamlStep {
	switch (step.type) {
		case StepType.WalletAction:
			return convertWalletActionStep(step);
		case StepType.Credential:
			return convertCredentialStep(step);
		case StepType.CustomCheck:
			return convertCustomCheckStep(step);
		case StepType.UseCaseVerification:
			return convertUseCaseVerificationStep(step);
		default:
			throw new Error(`Unknown step type`);
	}
}

const DEFAULT_DEEPLINK_STEP_ID = 'get-deeplink';

function convertWalletActionStep(step: WalletActionStep): YamlStep<'mobile-automation'> {
	const yamlStep: YamlStep<'mobile-automation'> = {
		use: 'mobile-automation',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			action_id: getPath(step.data.action),
			version_id: getPath(step.data.version),
			video: true
		}
	};

	const actionYaml = step.data.action.code;
	if (actionYaml.includes('${DL}') || actionYaml.includes('${deeplink}')) {
		yamlStep.with.parameters = {
			deeplink: '${{' + DEFAULT_DEEPLINK_STEP_ID + '.outputs}}'
		};
	}

	return yamlStep;
}

function convertCredentialStep(step: CredentialStep): YamlStep<'credential-offer'> {
	return {
		use: 'credential-offer',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			credential_id: getPath(step.data)
		}
	};
}

function convertCustomCheckStep(step: CustomCheckStep): YamlStep<'custom-check'> {
	return {
		use: 'custom-check',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			check_id: getPath(step.data)
		}
	};
}

function convertUseCaseVerificationStep(
	step: UseCaseVerificationStep
): YamlStep<'use-case-verification-deeplink'> {
	return {
		use: 'use-case-verification-deeplink',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			use_case_id: getPath(step.data)
		}
	};
}

function linkIds(steps: AnyYamlStep[]): AnyYamlStep[] {
	for (const [index, step] of steps.entries()) {
		if (!(step.use === 'mobile-automation')) continue;

		if (!step.with.parameters) continue;
		if (!('deeplink' in step.with.parameters)) continue;

		const previousStepId = steps
			.slice(0, index)
			.toReversed()
			.filter((s) => s.use != 'mobile-automation')
			.at(0)?.id;

		if (!previousStepId) continue;

		step.with.parameters.deeplink = step.with.parameters.deeplink.replace(
			DEFAULT_DEEPLINK_STEP_ID,
			previousStepId
		);
	}
	return steps;
}

/* YAML formatting */

export function formatYaml(yaml: string): string {
	return pipe(
		yaml,
		// Adding spaces
		addNewlineBefore('runtime:'),
		addNewlineBefore('steps:'),
		addNewlineBefore('  - use:'),
		// Correcting first step newline
		replaceWith('\n  - use:', (t) => t.replace('\n', ''), false)
	);
}

function addNewlineBefore(token: string, all = true) {
	return replaceWith(token, (token) => `\n${token}`, all);
}

function replaceWith(token: string, transform: (token: string) => string, all = true) {
	if (all) {
		return String.replaceAll(token, transform(token));
	} else {
		return String.replace(token, transform(token));
	}
}
