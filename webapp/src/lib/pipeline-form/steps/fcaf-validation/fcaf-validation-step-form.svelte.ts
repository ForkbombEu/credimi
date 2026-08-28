// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepByType, PipelineStepData } from '$lib/pipeline/types';

import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types';
import { parse } from 'yaml';

import Component from './fcaf-validation-step-form.svelte';

//

export type FCAFValidationStepData = PipelineStepData<PipelineStepByType<'fcaf-validation'>>;

export type FCAFValidationFormData = {
	yaml: string;
};

export class FCAFValidationStepForm extends BaseForm<
	FCAFValidationFormData,
	FCAFValidationStepForm
> {
	readonly Component = Component;

	data = $state<FCAFValidationFormData>({
		yaml: 'suite: wallet_solution/relying_party\ntest_ids: []\npipeline_outputs: {}\n'
	});

	constructor(opts?: InitFormOptions<FCAFValidationFormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...opts.initial };
		}
	}

	get validationError(): string | undefined {
		try {
			parseFCAFValidationConfiguration(this.data.yaml);
			return undefined;
		} catch (error) {
			return error instanceof Error ? error.message : String(error);
		}
	}

	get isValid(): boolean {
		return this.validationError === undefined;
	}

	canSave() {
		return this.isValid;
	}

	getSubmitData() {
		return this.isValid ? this.data : undefined;
	}

	submit() {
		this.commit();
	}
}

export function parseFCAFValidationConfiguration(yaml: string): FCAFValidationStepData {
	const value: unknown = parse(yaml);
	if (typeof value !== 'object' || value === null || Array.isArray(value)) {
		throw new Error('FCAF validation configuration must be a YAML object');
	}
	return value as FCAFValidationStepData;
}

export function getFCAFValidationTestIDs(yaml: string): string[] {
	const data = parseFCAFValidationConfiguration(yaml);
	if (Array.isArray(data.test_ids)) return data.test_ids;
	return data.test_id ? [data.test_id] : [];
}
