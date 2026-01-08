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
	type UtilityStep,
	type EmailStep,
	type DebugStep,
	type HttpRequestStep
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
		const emailStep = step as EmailStep;
		return {
			use: 'email',
			id: emailStep.id,
			continue_on_error: emailStep.continueOnError ?? false,
			with: {
				recipient: emailStep.data.recipient,
				...(emailStep.data.subject && { subject: emailStep.data.subject }),
				...(emailStep.data.body && { body: emailStep.data.body })
			}
		} as AnyYamlStep;
	} else if (step.type === StepType.Debug) {
		// Debug step: We don't have a dedicated debug activity in the registry,
		// so we create a no-op step that can be used as a breakpoint marker.
		// The step will succeed immediately without external dependencies.
		const debugStep = step as DebugStep;
		return {
			use: 'http-request',
			id: debugStep.id,
			continue_on_error: debugStep.continueOnError ?? false,
			with: {
				method: 'GET',
				url: 'http://localhost:1', // Minimal localhost request that will timeout safely
				timeout: '1' // 1 second timeout to fail fast
			},
			metadata: {
				debug: true,
				description: 'Debug breakpoint - this step is expected to fail quickly'
			}
		} as AnyYamlStep;
	} else if (step.type === StepType.HttpRequest) {
		const httpStep = step as HttpRequestStep;
		return {
			use: 'http-request',
			id: httpStep.id,
			continue_on_error: httpStep.continueOnError ?? false,
			with: {
				method: httpStep.data.method,
				url: httpStep.data.url,
				...(httpStep.data.headers && Object.keys(httpStep.data.headers).length > 0
					? { headers: httpStep.data.headers }
					: {}),
				...(httpStep.data.body && { body: httpStep.data.body })
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
