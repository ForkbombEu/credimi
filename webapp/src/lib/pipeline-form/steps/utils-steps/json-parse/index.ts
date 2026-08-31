// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { TypedConfig } from '$pipeline-form/steps/types';

import { Braces } from '@lucide/svelte';
import config from '$config';

import { m } from '@/i18n';

import { JsonParseStepForm, type JsonParseFormData } from './json-parse-step-form.svelte.js';

//

const jsonParseEntity = {
	slug: 'json-parse',
	icon: Braces,
	labels: {
		singular: m.JSON_Parse(),
		plural: m.JSON_Parse()
	},
	classes: {
		bg: 'bg-[hsl(var(--gray-background))]',
		text: 'text-[hsl(var(--gray-foreground))]',
		border: 'border-[hsl(var(--gray-outline))]'
	}
};

export const jsonParseStepConfig: TypedConfig<'json-parse', JsonParseFormData> = {
	use: 'json-parse',
	docsUrl: config.externalLinks.docs.pipeline.utils,

	display: jsonParseEntity,

	initForm: (opts) => new JsonParseStepForm(opts),

	serialize: ({ rawJSON, struct_type }) => ({
		rawJSON,
		struct_type
	}),

	deserialize: async ({ rawJSON, struct_type }) => ({
		rawJSON: rawJSON == null || rawJSON === '' ? '{}' : rawJSON,
		struct_type: struct_type == null || struct_type === '' ? 'map' : struct_type
	}),

	cardData: ({ struct_type }) => ({
		title: m.JSON_Parse(),
		copyText: struct_type
	}),

	makeId: ({ struct_type }) => `json-parse-${struct_type || 'map'}`
};
