// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { Array, pipe, String } from 'effect';

import type { HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition as Pipeline } from './types.generated';

import {
	StepType,
	type BuilderStep,
	type ConformanceCheckStep,
	type MarketplaceItemStep,
	type WalletActionStep,
	type UtilityStep
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
	if (step.type === StepType.WalletAction) {
		return convertWalletActionStep(step);
	} else if (step.type === StepType.ConformanceCheck) {
		return convertConformanceCheckStep(step);
	} else if (
		step.type === StepType.Email ||
		step.type === StepType.Debug ||
		step.type === StepType.HttpRequest
	) {
		return convertUtilityStep(step);
	} else {
		return convertMarketplaceItemStep(step);
	}
}

const DEEPLINK_STEP_ID_PLACEHOLDER = 'get-deeplink';

function convertWalletActionStep(step: WalletActionStep): YamlStep<'mobile-automation'> {
	const yamlStep: YamlStep<'mobile-automation'> = {
		use: 'mobile-automation',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			action_id: getPath(step.data.action),
			version_id: getPath(step.data.version)
		}
	};

	const actionYaml = step.data.action.code;
	if (actionYaml.includes('${DL}') || actionYaml.includes('${deeplink}')) {
		yamlStep.with.parameters = {
			deeplink: '${{' + DEEPLINK_STEP_ID_PLACEHOLDER + '}}'
		};
	}

	return yamlStep;
}

function convertMarketplaceItemStep(step: MarketplaceItemStep) {
	switch (step.type) {
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

function convertCredentialStep(step: MarketplaceItemStep): YamlStep<'credential-offer'> {
	return {
		use: 'credential-offer',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			credential_id: getPath(step.data)
		}
	};
}

function convertCustomCheckStep(step: MarketplaceItemStep): YamlStep<'custom-check'> {
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
	step: MarketplaceItemStep
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

function convertConformanceCheckStep(step: ConformanceCheckStep): YamlStep<'conformance-check'> {
	return {
		use: 'conformance-check',
		id: step.id,
		continue_on_error: step.continueOnError ?? false,
		with: {
			check_id: step.data.checkId
		}
	};
}

function convertUtilityStep(step: UtilityStep): AnyYamlStep {
	if (step.type === StepType.Email) {
		return {
			use: 'email',
			id: step.id,
			continue_on_error: step.continueOnError ?? false,
			with: {
				payload: {
					recipient: step.data.recipient,
					...(step.data.subject && { subject: step.data.subject }),
					...(step.data.body && { body: step.data.body })
				}
			}
		} as AnyYamlStep;
	} else if (step.type === StepType.Debug) {
		// Debug step uses the pipeline debug activity which is handled internally
		// We'll use http-request as a placeholder since there's no direct debug step in the registry
		// The backend handles debug through the runtime.debug flag
		return {
			use: 'http-request',
			id: step.id,
			continue_on_error: step.continueOnError ?? false,
			with: {
				payload: {
					method: 'GET',
					url: 'https://httpbin.org/status/200' // Simple debug endpoint
				}
			},
			metadata: {
				debug: true
			}
		} as AnyYamlStep;
	} else if (step.type === StepType.HttpRequest) {
		return {
			use: 'http-request',
			id: step.id,
			continue_on_error: step.continueOnError ?? false,
			with: {
				payload: {
					method: step.data.method,
					url: step.data.url,
					...(step.data.headers && Object.keys(step.data.headers).length > 0
						? { headers: step.data.headers }
						: {}),
					...(step.data.body && { body: step.data.body })
				}
			}
		} as AnyYamlStep;
	}
	throw new Error(`Unknown utility step type: ${step.type}`);
}

function linkIds(steps: AnyYamlStep[]): AnyYamlStep[] {
	for (const [index, step] of steps.entries()) {
		if (!(step.use === 'mobile-automation')) continue;

		if (!step.with.parameters) continue;
		if (!('deeplink' in step.with.parameters)) continue;

		const previousStep = steps
			.slice(0, index)
			.toReversed()
			.filter((s) => s.use != 'mobile-automation')
			.at(0);

		if (!previousStep) continue;

		let deeplinkPath = '.outputs';
		if (previousStep.use === 'conformance-check') {
			deeplinkPath += '.deeplink';
		}

		step.with.parameters.deeplink = step.with.parameters.deeplink.replace(
			DEEPLINK_STEP_ID_PLACEHOLDER,
			previousStep.id + deeplinkPath
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
