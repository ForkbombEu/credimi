// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Globe, Mail } from 'lucide-svelte';

import { m } from '@/i18n';

import type { TypedConfig } from '../types';

import { EmailStepForm, type EmailFormData } from './email-step-form.svelte.js';
import { HttpRequestStepForm, type HttpRequestFormData } from './http-request-step-form.svelte.js';

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

	display: {
		...utilsEntity,
		icon: Mail,
		labels: {
			singular: m.Email(),
			plural: m.Email()
		}
	},

	initForm: () => new EmailStepForm(),

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

	display: {
		...utilsEntity,
		icon: Globe,
		labels: {
			singular: m.HTTP_Request(),
			plural: m.HTTP_Request()
		}
	},

	initForm: () => new HttpRequestStepForm(),

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
		const urlPath = new URL(data.url || 'unknown').host;
		return `http-${method}-${urlPath}`;
	}
};
