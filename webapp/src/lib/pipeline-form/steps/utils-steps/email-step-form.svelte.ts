// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { appName } from '@/brand';

import { BaseForm } from '../types';
import Component from './email-step-form.svelte';
import { formatPlaceholder as fmt, Placeholder } from './placeholders/utils';

//

export class EmailStepForm extends BaseForm<EmailFormData, EmailStepForm> {
	readonly Component = Component;

	data = $state<EmailFormData>({
		recipient: '',
		subject: `${appName} | Pipeline "${fmt(Placeholder.PIPELINE_NAME)}" result: ${fmt(Placeholder.RESULT)}`,
		body: defaultBody()
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

export function defaultBody() {
	return `Hi!

Your pipeline "${fmt(Placeholder.PIPELINE_NAME)}" has completed execution with status "${fmt(Placeholder.RESULT)}" at ${fmt(Placeholder.DATE)}.

View the full pipeline execution details here:
${fmt(Placeholder.PIPELINE_URL)}

Best regards,
${appName} team`;
}
