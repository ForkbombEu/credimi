// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CustomChecksResponse } from '@/pocketbase/types';

import { describe, expect, it, vi } from 'vitest';

vi.mock('./custom-integration-step-form.svelte', () => ({ default: class {} }));

vi.mock('@/components/json-schema-form', () => ({
	createJsonSchemaForm: vi.fn(() => ({}))
}));

vi.mock('@sjsf/form', () => ({
	getValueSnapshot: vi.fn(() => ({ apiKey: 'abc' })),
	validate: vi.fn(() => ({ errors: [{ message: 'required' }] }))
}));

import { validate } from '@sjsf/form';

import { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';

const integrationNoSchema = {
	id: 'ci1',
	name: 'Plain',
	input_json_schema: null
} as CustomChecksResponse;

const integrationWithSchema = {
	id: 'ci2',
	name: 'With Schema',
	input_json_schema: { type: 'object', properties: { apiKey: { type: 'string' } } },
	input_json_sample: { apiKey: '' }
} as CustomChecksResponse;

describe('CustomIntegrationStepForm', () => {
	it('selectIntegration auto-commits on add when no schema', () => {
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm({ intent: 'add' });
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationNoSchema);
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].integration).toBe(integrationNoSchema);
	});

	it('selectIntegration does not auto-commit on add when schema exists', () => {
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm({ intent: 'add' });
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationWithSchema);
		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.state).toBe('configure');
	});

	it('canSave is false when schema invalid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [{ message: 'required' }] } as never);
		const form = new CustomIntegrationStepForm({
			intent: 'edit',
			initial: { integration: integrationWithSchema, config: {} }
		});
		expect(form.canSave()).toBe(false);
	});

	it('canSave is true when schema valid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const form = new CustomIntegrationStepForm({
			intent: 'edit',
			initial: { integration: integrationWithSchema, config: { apiKey: 'abc' } }
		});
		expect(form.canSave()).toBe(true);
	});

	it('discardIntegration clears selection and schema form', () => {
		const form = new CustomIntegrationStepForm({
			intent: 'edit',
			initial: { integration: integrationWithSchema }
		});
		form.discardIntegration();
		expect(form.data.integration).toBeUndefined();
		expect(form.jsonSchemaForm).toBeUndefined();
		expect(form.state).toBe('select-integration');
	});

	it('submit commits valid schema config', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm({
			intent: 'add',
			initial: { integration: integrationWithSchema }
		});
		form.onSubmit(onSubmit);
		form.submit();
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].config).toEqual({ apiKey: 'abc' });
	});
});
