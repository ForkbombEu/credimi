// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Mail, Globe } from 'lucide-svelte';

import { m } from '@/i18n';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
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
		subject: data.subject || undefined,
		body: data.body || undefined,
		sender: data.sender || undefined
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
		copyText: data.recipient,
		avatar: undefined
	}),

	makeId: (data) => {
		const username = data.recipient.split('@')[0] || 'email';
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
		let bodyValue: any = undefined;
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
			body: bodyValue,
			headers: data.headers || undefined
		};
	},

	deserialize: async (data) => {
		let bodyString = '';
		if (data.body) {
			bodyString = typeof data.body === 'string' 
				? data.body 
				: JSON.stringify(data.body, null, 2);
		}
		return {
			method: data.method,
			url: data.url,
			body: bodyString,
			headers: data.headers || undefined
		};
	},

	cardData: (data) => ({
		title: `${data.method} Request`,
		copyText: data.url,
		avatar: undefined
	}),

	makeId: (data) => `http-${data.method.toLowerCase()}-${getLastPathSegment(data.url)}`
};
