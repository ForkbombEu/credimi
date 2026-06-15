// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getValueSnapshot, validate } from '@sjsf/form';
import { getPath } from '$lib/utils';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';

import { BaseForm, type InitFormOptions } from '../types';
import { resolveInitialConfig, setStoredConfig } from './config-storage.js';
import Component from './custom-integration-step-form.svelte';

export type CustomIntegrationStepFormData = {
	integration: CustomChecksResponse;
	parameters?: Record<string, unknown>;
};

type FormState = 'select-integration' | 'configure' | 'ready';

export class CustomIntegrationStepForm extends BaseForm<
	CustomIntegrationStepFormData,
	CustomIntegrationStepForm
> {
	readonly Component = Component;

	data = $state<Partial<CustomIntegrationStepFormData>>({});
	jsonSchemaForm = $state<JsonSchemaForm | undefined>(undefined);

	constructor(opts?: InitFormOptions<CustomIntegrationStepFormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...opts.initial };
			this.initJsonSchemaForm(opts.initial.integration, opts.initial.parameters);
		}
	}

	hasSchema = $derived(Boolean(this.data.integration?.input_json_schema));

	isSchemaValid = $derived.by(() => {
		if (!this.hasSchema) return true;
		const jsonSchemaForm = this.jsonSchemaForm;
		if (!jsonSchemaForm) return false;
		getValueSnapshot(jsonSchemaForm);
		return (validate(jsonSchemaForm).errors ?? []).length === 0;
	});

	isValid = $derived.by(() => {
		if (!this.data.integration) return false;
		return this.isSchemaValid;
	});

	state: FormState = $derived.by(() => {
		if (!this.data.integration) return 'select-integration';
		if (this.hasSchema && !this.isSchemaValid) return 'configure';
		return 'ready';
	});

	canSave() {
		return this.isValid;
	}

	getSubmitData(): CustomIntegrationStepFormData | undefined {
		if (this.state !== 'ready' || !this.data.integration) return undefined;
		const parameters = this.jsonSchemaForm
			? (getValueSnapshot(this.jsonSchemaForm) as Record<string, unknown>)
			: undefined;
		return {
			integration: this.data.integration,
			parameters
		};
	}

	commit(data?: CustomIntegrationStepFormData) {
		const payload = data ?? this.getSubmitData();
		if (payload === undefined) return;
		super.commit(payload);
		if (payload.parameters && payload.integration) {
			setStoredConfig(getPath(payload.integration, true), payload.parameters);
		}
	}

	selectIntegration(integration: CustomChecksResponse) {
		this.data.integration = integration;
		this.data.parameters = undefined;
		this.initJsonSchemaForm(integration);
		if (this.intent === 'add' && !this.hasSchema) {
			const payload = this.getSubmitData();
			if (payload) this.commit(payload);
		}
	}

	discardIntegration() {
		this.data.integration = undefined;
		this.data.parameters = undefined;
		this.jsonSchemaForm = undefined;
	}

	submit() {
		this.commit();
	}

	private initJsonSchemaForm(
		integration: CustomChecksResponse,
		initialParameters?: Record<string, unknown>
	) {
		const schema = integration.input_json_schema;
		if (schema) {
			this.jsonSchemaForm = createJsonSchemaForm(schema as object, {
				hideTitle: true,
				initialValue: resolveInitialConfig(integration, initialParameters)
			});
		} else {
			this.jsonSchemaForm = undefined;
		}
	}
}
