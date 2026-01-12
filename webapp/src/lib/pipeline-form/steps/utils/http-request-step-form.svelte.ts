// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseDataForm } from '../types';
import Component from './http-request-step-form.svelte';

//

export class HttpRequestStepForm extends BaseDataForm<HttpRequestFormData, HttpRequestStepForm> {
	readonly Component = Component;

	data = $state<HttpRequestFormData>({
		method: 'GET',
		url: ''
	});

	get isValid(): boolean {
		return this.data.url.trim() !== '' && this.data.method.trim() !== '';
	}

	submit() {
		if (this.isValid) {
			this.handleSubmit(this.data);
		}
	}
}

//

export type HttpRequestFormData = {
	method: string;
	url: string;
	body?: string;
	headers?: Record<string, string>;
};
