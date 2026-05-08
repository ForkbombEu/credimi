// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseForm } from '../types';
import Component from './http-request-step-form.svelte';
import { formatPlaceholder, Placeholder } from './placeholders/utils';

//

export class HttpRequestStepForm extends BaseForm<HttpRequestFormData, HttpRequestStepForm> {
	readonly Component = Component;

	data = $state<HttpRequestFormData>({
		method: 'POST',
		url: '',
		body: defaultBody()
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
};

function defaultBody() {
	return JSON.stringify(
		Object.fromEntries(
			Object.values(Placeholder).map((placeholder) => [
				placeholder,
				formatPlaceholder(placeholder)
			])
		),
		null,
		2
	);
}
