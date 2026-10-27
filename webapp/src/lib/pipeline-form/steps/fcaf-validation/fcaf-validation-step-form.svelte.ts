// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepByType, PipelineStepData } from '$lib/pipeline/types';

import { createQuery } from '@tanstack/svelte-query';
import { FCAF } from '$lib';
import { queryClient } from '$lib/query-client';
import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types';
import { parse, stringify } from 'yaml';

import Component from './fcaf-validation-step-form.svelte';

//

export type FCAFValidationStepData = PipelineStepData<PipelineStepByType<'fcaf-validation'>>;

export type FCAFValidationFormData = {
	yaml: string;
};

/** Explicit client: this form is constructed inside `$effect.root()`, not component init. */
const queryClientAccessor = () => queryClient;

function defaultFCAFValidationYaml(): string {
	return stringify({
		suite: FCAF.SUITE,
		test_ids: [],
		pipeline_outputs: {}
	});
}

function filterPipelineOutputsFor(suite: string, testIds: string[]): Record<string, unknown> {
	if (testIds.length === 0) return {};
	// Full defaults map from the suite's aggregate pipeline — validators ignore unused keys.
	return { ...(FCAF.PIPELINE_OUTPUTS_BY_SUITE[suite] ?? {}) };
}

export class FCAFValidationStepForm extends BaseForm<
	FCAFValidationFormData,
	FCAFValidationStepForm
> {
	readonly Component = Component;

	fcafTests = createQuery(
		() => ({
			queryKey: ['conformance-checks', 'fcaf'] as const,
			queryFn: async () => {
				const result = await FCAF.getFcafTests({ surface: 'pipeline' });
				if (result instanceof Error) throw result;
				return result;
			}
		}),
		queryClientAccessor
	);

	data = $state<FCAFValidationFormData>({
		yaml: defaultFCAFValidationYaml()
	});

	constructor(opts?: InitFormOptions<FCAFValidationFormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...opts.initial };
		}
	}

	/** The configured suite, or the default suite while the YAML names none or is invalid. */
	get suite(): string {
		try {
			const { suite } = parseFCAFValidationConfiguration(this.data.yaml);
			return typeof suite === 'string' && suite !== '' ? suite : FCAF.SUITE;
		} catch {
			return FCAF.SUITE;
		}
	}

	/** Catalog tests of the configured suite: a validation step validates one suite. */
	get availableTests(): FCAF.TestCatalogEntry[] {
		const suite = this.suite;
		return (this.fcafTests.data ?? []).filter((test) => test.suite === suite);
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

	/**
	 * Writes the selection. Once the catalog is loaded, ids outside the configured
	 * suite are dropped: they remain after the suite is changed in the YAML, and
	 * the activity, which loads only that suite's catalog, would reject them.
	 */
	setTestIds(testIds: string[]) {
		const config = parseFCAFValidationConfiguration(this.data.yaml) as Record<string, unknown>;
		const suite = this.suite;
		const ids = this.fcafTests.data
			? testIds.filter((id) => this.availableTests.some((test) => test.id === id))
			: testIds;
		config.test_ids = [...ids];
		config.pipeline_outputs = filterPipelineOutputsFor(suite, ids);
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
