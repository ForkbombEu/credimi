// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { nanoid } from 'nanoid';
import { stringify } from 'yaml';

import type { HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition as PipelineSchema } from './types.generated';

import {
	StepType,
	type BuilderStep,
	type CredentialStep,
	type CustomCheckStep,
	type UseCaseVerificationStep,
	type WalletActionStep
} from './types';

// Types

type YamlSteps = NonNullable<PipelineSchema['steps']>[number];
type YamlStepId = YamlSteps['use'];
type Step<Id extends YamlStepId> = Extract<YamlSteps, { use: Id }>;
type AnyYamlStep = Step<YamlStepId>;

//

export function buildYaml(steps: BuilderStep[]): string {
	const convertedSteps = steps.map(convertStep);
	return stringify(linkIds(convertedSteps));
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

function convertWalletActionStep(step: WalletActionStep): Step<'mobile-automation'> {
	return {
		use: 'mobile-automation',
		id: generateStepId(step.data.action.canonified_name),
		continue_on_error: step.continueOnError ?? false,
		with: {
			action_id: step.data.action.canonified_name,
			version_id: step.data.version.canonified_tag,
			video: true,
			parameters: {
				deeplink: '${{' + DEFAULT_DEEPLINK_STEP_ID + '.outputs}}'
			}
		}
	};
}

function convertCredentialStep(step: CredentialStep): Step<'credential-offer'> {
	return {
		use: 'credential-offer',
		id: generateStepId(step.data.canonified_name),
		continue_on_error: step.continueOnError ?? false,
		with: {
			credential_id: step.path
		}
	};
}

function convertCustomCheckStep(step: CustomCheckStep): Step<'custom-check'> {
	return {
		use: 'custom-check',
		id: generateStepId(step.data.canonified_name),
		continue_on_error: step.continueOnError ?? false,
		with: {
			check_id: step.path
		}
	};
}

function convertUseCaseVerificationStep(
	step: UseCaseVerificationStep
): Step<'use-case-verification-deeplink'> {
	return {
		use: 'use-case-verification-deeplink',
		id: generateStepId(step.data.canonified_name),
		continue_on_error: step.continueOnError ?? false,
		with: {
			use_case_id: step.path
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

function generateStepId(text: string): string {
	return text + '__' + nanoid(5);
}
