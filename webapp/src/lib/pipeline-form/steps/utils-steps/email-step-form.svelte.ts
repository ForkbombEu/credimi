// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseDataForm } from '../types';
import Component from './email-step-form.svelte';

//

export class EmailStepForm extends BaseDataForm<EmailFormData, EmailStepForm> {
	readonly Component = Component;

	data = $state<EmailFormData>({
		recipient: '',
		subject: '',
		body: ''
	});

	get isValid(): boolean {
		return this.data.recipient.trim() !== '';
	}

	submit() {
		if (this.isValid) {
			this.handleSubmit(this.data);
		}
	}
}

//

export type EmailFormData = {
	recipient: string;
	subject: string;
	body: string;
	sender?: string;
};
