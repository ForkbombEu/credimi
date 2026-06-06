// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getValueSnapshot, validate } from '@sjsf/form';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';

import { BaseForm, type InitFormOptions } from '../types';
import Component from './custom-integration-step-form.svelte';

export type CustomIntegrationStepFormData = {
	integration: CustomChecksResponse;
	config?: Record<string, unknown>;
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
			this.initJsonSchemaForm(opts.initial.integration, opts.initial.config);
		}
	}

	hasSchema = $derived(Boolean(this.data.integration?.input_json_schema));

	isSchemaValid = $derived.by(() => {
		if (!this.hasSchema) return true;
		if (!this.jsonSchemaForm) return false;
		return (validate(this.jsonSchemaForm).errors ?? []).length === 0;
	});

	state: FormState = $derived.by(() => {
		if (!this.data.integration) return 'select-integration';
		if (this.hasSchema && !this.isSchemaValid) return 'configure';
		return 'ready';
	});

	canSave() {
		return this.state === 'ready';
	}

	getSubmitData(): CustomIntegrationStepFormData | undefined {
		if (this.state !== 'ready' || !this.data.integration) return undefined;
		const config = this.jsonSchemaForm
			? (getValueSnapshot(this.jsonSchemaForm) as Record<string, unknown>)
			: undefined;
		return {
			integration: this.data.integration,
			config
		};
	}

	selectIntegration(integration: CustomChecksResponse) {
		this.data.integration = integration;
		this.data.config = undefined;
		this.initJsonSchemaForm(integration);
		if (this.intent === 'add' && !this.hasSchema) {
			const payload = this.getSubmitData();
			if (payload) this.commit(payload);
		}
	}

	discardIntegration() {
		this.data.integration = undefined;
		this.data.config = undefined;
		this.jsonSchemaForm = undefined;
	}

	submit() {
		this.commit();
	}

	private initJsonSchemaForm(integration: CustomChecksResponse, initialConfig?: Record<string, unknown>) {
		const schema = integration.input_json_schema;
		if (schema) {
			this.jsonSchemaForm = createJsonSchemaForm(schema as object, {
				hideTitle: true,
				initialValue: initialConfig ?? integration.input_json_sample ?? undefined
			});
		} else {
			this.jsonSchemaForm = undefined;
		}
	}
}
