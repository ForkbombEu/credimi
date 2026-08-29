// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepByType, PipelineStepData } from '$lib/pipeline/types';

import {
	FCAF_PIPELINE_OUTPUTS,
	FCAF_SUITE,
	FCAF_TESTS,
	type FCAFTestCatalogEntry
} from '$lib/fcaf/tests.generated.js';
import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types';
import { parse, stringify } from 'yaml';

import Component from './fcaf-validation-step-form.svelte';

//

export type FCAFValidationStepData = PipelineStepData<PipelineStepByType<'fcaf-validation'>>;

export type FCAFValidationFormData = {
	yaml: string;
};

function defaultFCAFValidationYaml(): string {
	return stringify({
		suite: FCAF_SUITE,
		test_ids: [],
		pipeline_outputs: {}
	});
}

function filterPipelineOutputsFor(testIds: string[]): Record<string, unknown> {
	const needed: string[] = [];
	for (const test of FCAF_TESTS) {
		if (!testIds.includes(test.id)) continue;
		for (const source of test.sources) {
			if (!needed.includes(source)) needed.push(source);
		}
	}

	const outputs = FCAF_PIPELINE_OUTPUTS as Record<string, unknown>;
	const filtered: Record<string, unknown> = {};
	for (const source of needed) {
		if (source in outputs) filtered[source] = outputs[source];
	}
	return filtered;
}

export class FCAFValidationStepForm extends BaseForm<
	FCAFValidationFormData,
	FCAFValidationStepForm
> {
	readonly Component = Component;

	readonly availableTests: FCAFTestCatalogEntry[] = FCAF_TESTS;

	data = $state<FCAFValidationFormData>({
		yaml: defaultFCAFValidationYaml()
	});

	constructor(opts?: InitFormOptions<FCAFValidationFormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...opts.initial };
		}
	}

	get selectedTestIds(): string[] {
		try {
			return getFCAFValidationTestIDs(this.data.yaml);
		} catch {
			return [];
		}
	}

	get pipelineOutputsCount(): number {
		try {
			const config = parseFCAFValidationConfiguration(this.data.yaml) as Record<
				string,
				unknown
			>;
			const outputs = config.pipeline_outputs;
			return outputs && typeof outputs === 'object' ? Object.keys(outputs).length : 0;
		} catch {
			return 0;
		}
	}

	setTestIds(testIds: string[]) {
		const config = parseFCAFValidationConfiguration(this.data.yaml) as Record<string, unknown>;
		config.test_ids = [...testIds];
		config.pipeline_outputs = filterPipelineOutputsFor(testIds);
		delete config.test_id;
		this.data.yaml = stringify(config);
	}

	toggleTestId(testId: string) {
		const ids = this.selectedTestIds;
		this.setTestIds(
			ids.includes(testId) ? ids.filter((id) => id !== testId) : [...ids, testId]
		);
	}

	selectAllTestIds() {
		this.setTestIds(this.availableTests.map((test) => test.id));
	}

	clearTestIds() {
		this.setTestIds([]);
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
