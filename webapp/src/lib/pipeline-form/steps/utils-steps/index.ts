// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { TypedConfig } from '$pipeline-form/steps/types';

import { Globe, Mail } from '@lucide/svelte';
import config from '$config';

import { m } from '@/i18n';

import { EmailStepForm, type EmailFormData } from './email-step-form.svelte.js';
import { HttpRequestStepForm, type HttpRequestFormData } from './http-request-step-form.svelte.js';

export { jsonParseStepConfig } from './json-parse';

//

const utilsEntity = {
	slug: 'utils',
	icon: Mail,
	labels: {
		singular: m.Utils(),
		plural: m.Utils()
	},
	classes: {
		bg: 'bg-[hsl(var(--gray-background))]',
		text: 'text-[hsl(var(--gray-foreground))]',
		border: 'border-[hsl(var(--gray-outline))]'
	}
};

//

export const emailStepConfig: TypedConfig<'email', EmailFormData> = {
	use: 'email',
	docsUrl: config.externalLinks.docs.pipeline.utils,

	display: {
		...utilsEntity,
		icon: Mail,
		labels: {
			singular: m.Email(),
			plural: m.Email()
		}
	},

	initForm: (opts) => new EmailStepForm(opts),

	serialize: (data) => ({
		recipient: data.recipient,
		subject: data.subject,
		body: data.body,
		sender: data.sender || ''
	}),

	deserialize: async (data) => {
		return {
			recipient: data.recipient,
			subject: data.subject || '',
			body: data.body || '',
			sender: data.sender || ''
		};
	},

	cardData: (data) => ({
		title: m.Email(),
		copyText: data.recipient
	}),

	makeId: (data) => {
		const username = (data.recipient || 'email').split('@')[0] || 'email';
		return `email-${username}`;
	}
};

//

export const httpRequestStepConfig: TypedConfig<'http-request', HttpRequestFormData> = {
	use: 'http-request',
	docsUrl: config.externalLinks.docs.pipeline.utils,

	display: {
		...utilsEntity,
		icon: Globe,
		labels: {
			singular: m.HTTP_Request(),
			plural: m.HTTP_Request()
		}
	},

	initForm: (opts) => new HttpRequestStepForm(opts),

	serialize: (data) => {
		let bodyValue: unknown = undefined;
		if (data.body && data.body.trim()) {
			try {
				bodyValue = JSON.parse(data.body);
			} catch {
				bodyValue = data.body;
			}
		}
		return {
			method: data.method,
			url: data.url,
			body: bodyValue
		};
	},

	deserialize: async (data) => {
		let bodyString = '';
		if (data.body) {
			bodyString =
				typeof data.body === 'string' ? data.body : JSON.stringify(data.body, null, 2);
		}
		return {
			method: data.method,
			url: data.url,
			body: bodyString
		};
	},

	cardData: (data) => ({
		title: `${data.method} Request`,
		copyText: data.url,
		meta: {
			url: data.url
		}
	}),

	makeId: (data) => {
		const method = (data.method || 'request').toLowerCase();
		return `http-${method}-${httpRequestIdHost(data.url)}`;
	}
};

/** Host fragment for step ids; template/relative URLs must not throw. */
export function httpRequestIdHost(url: string | undefined): string {
	const raw = url?.trim() || 'unknown';
	if (URL.canParse(raw)) {
		return new URL(raw).host || 'unknown';
	}
	const slug = raw
		.replace(/[^a-zA-Z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '')
		.slice(0, 48);
	return slug || 'unknown';
}
