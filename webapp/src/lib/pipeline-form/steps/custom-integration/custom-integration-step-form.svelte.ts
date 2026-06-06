// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseForm, type InitFormOptions } from '../types';
import Component from './custom-integration-step-form.svelte';

export type CustomIntegrationStepFormData = {
	integration: import('@/pocketbase/types').CustomChecksResponse;
	config?: Record<string, unknown>;
};

export class CustomIntegrationStepForm extends BaseForm<
	CustomIntegrationStepFormData,
	CustomIntegrationStepForm
> {
	readonly Component = Component;
	canSave() {
		return false;
	}
	getSubmitData() {
		return undefined;
	}
	constructor(_opts?: InitFormOptions<CustomIntegrationStepFormData>) {
		super(_opts);
	}
}
