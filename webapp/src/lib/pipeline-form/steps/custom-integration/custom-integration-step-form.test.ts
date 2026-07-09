// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { CustomChecksResponse } from '@/pocketbase/types';

vi.mock('./custom-integration-step-form.svelte', () => ({ default: class {} }));

vi.mock('@/components/json-schema-form', () => ({
	createJsonSchemaForm: vi.fn(() => ({}))
}));

vi.mock('@sjsf/form', () => ({
	getValueSnapshot: vi.fn(() => ({ apiKey: 'abc' })),
	validate: vi.fn(() => ({ errors: [{ message: 'required' }] }))
}));

vi.mock('$lib/utils', () => ({
	getPath: vi.fn((record: { canonified_name?: string }, trim?: boolean) =>
		trim ? (record.canonified_name ?? '') : (record.canonified_name ?? '')
	)
}));

vi.mock('./config-storage.js', () => ({
	getStoredConfig: vi.fn(),
	resolveInitialConfig: vi.fn((integration, explicitParameters) => {
		if (explicitParameters !== undefined) return explicitParameters;
		return undefined;
	}),
	setStoredConfig: vi.fn()
}));

import { validate } from '@sjsf/form';
import { createInitFormOptions } from '$pipeline-form/steps/init-form-options.test-utils.js';

import { createJsonSchemaForm } from '@/components/json-schema-form';

import { resolveInitialConfig, setStoredConfig } from './config-storage.js';
import { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';

const integrationNoSchema = {
	id: 'ci1',
	name: 'Plain',
	input_json_schema: null
} as CustomChecksResponse;

const integrationWithSchema = {
	id: 'ci2',
	name: 'With Schema',
	canonified_name: 'org/with-schema',
	input_json_schema: { type: 'object', properties: { apiKey: { type: 'string' } } },
	input_json_sample: { apiKey: 'sample-value' }
} as CustomChecksResponse;

describe('CustomIntegrationStepForm', () => {
	beforeEach(() => {
		vi.mocked(validate).mockReset();
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		vi.mocked(createJsonSchemaForm).mockClear();
		vi.mocked(resolveInitialConfig).mockReset();
		vi.mocked(resolveInitialConfig).mockImplementation((_integration, explicitParameters) => {
			if (explicitParameters !== undefined) return explicitParameters;
			return undefined;
		});
		vi.mocked(setStoredConfig).mockReset();
	});

	it('selectIntegration auto-commits on add when no schema', () => {
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm(createInitFormOptions({ intent: 'add' }));
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationNoSchema);
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].integration).toBe(integrationNoSchema);
	});

	it('selectIntegration does not auto-commit on add when schema exists', () => {
		vi.mocked(validate).mockReturnValue({ errors: [{ message: 'required' }] } as never);
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm(createInitFormOptions({ intent: 'add' }));
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationWithSchema);
		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.state).toBe('configure');
	});

	it('selectIntegration does not auto-commit on edit intent', () => {
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm(createInitFormOptions({ intent: 'edit' }));
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationNoSchema);
		expect(onSubmit).not.toHaveBeenCalled();
	});

	it('canSave is false when schema invalid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [{ message: 'required' }] } as never);
		const form = new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'edit',
				initial: { integration: integrationWithSchema, parameters: {} }
			})
		);
		expect(form.canSave()).toBe(false);
		expect(form.isValid).toBe(false);
	});

	it('canSave is true when schema valid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const form = new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'edit',
				initial: { integration: integrationWithSchema, parameters: { apiKey: 'abc' } }
			})
		);
		expect(form.canSave()).toBe(true);
		expect(form.isValid).toBe(true);
	});

	it('discardIntegration clears selection and schema form', () => {
		const form = new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'edit',
				initial: { integration: integrationWithSchema }
			})
		);
		form.discardIntegration();
		expect(form.data.integration).toBeUndefined();
		expect(form.jsonSchemaForm).toBeUndefined();
		expect(form.state).toBe('select-integration');
	});

	it('submit commits valid schema parameters', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'add',
				initial: { integration: integrationWithSchema }
			})
		);
		form.onSubmit(onSubmit);
		form.submit();
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].parameters).toEqual({ apiKey: 'abc' });
	});

	it('selectIntegration does not pass input_json_sample to createJsonSchemaForm', () => {
		const form = new CustomIntegrationStepForm(createInitFormOptions({ intent: 'add' }));
		form.selectIntegration(integrationWithSchema);
		expect(createJsonSchemaForm).toHaveBeenCalledWith(integrationWithSchema.input_json_schema, {
			hideTitle: true,
			initialValue: undefined
		});
	});

	it('selectIntegration uses resolveInitialConfig without explicit parameters', () => {
		const form = new CustomIntegrationStepForm(createInitFormOptions({ intent: 'add' }));
		form.selectIntegration(integrationWithSchema);
		expect(resolveInitialConfig).toHaveBeenCalledWith(integrationWithSchema, undefined);
	});

	it('constructor passes explicit parameters to resolveInitialConfig in edit mode', () => {
		vi.mocked(resolveInitialConfig).mockReturnValue({ apiKey: 'from-yaml' });
		new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'edit',
				initial: {
					integration: integrationWithSchema,
					parameters: { apiKey: 'from-yaml' }
				}
			})
		);
		expect(resolveInitialConfig).toHaveBeenCalledWith(integrationWithSchema, {
			apiKey: 'from-yaml'
		});
	});

	it('commit persists parameters to localStorage', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'add',
				initial: { integration: integrationWithSchema }
			})
		);
		form.onSubmit(onSubmit);
		form.commit();
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(setStoredConfig).toHaveBeenCalledWith('org/with-schema', { apiKey: 'abc' });
	});

	it('commit does not persist when payload is invalid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [{ message: 'required' }] } as never);
		const form = new CustomIntegrationStepForm(
			createInitFormOptions({
				intent: 'add',
				initial: { integration: integrationWithSchema }
			})
		);
		form.commit();
		expect(setStoredConfig).not.toHaveBeenCalled();
	});
});
