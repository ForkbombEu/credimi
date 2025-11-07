// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ActivityOptions } from '$pipeline-form/pipeline.types.generated';
import PipelineSchema from '$root/schemas/pipeline/pipeline_schema.json';
import { createForm } from '@/forms';
import { getExceptionMessage } from '@/utils/errors';
import Ajv from 'ajv';
import type { SuperForm } from 'sveltekit-superforms';
import { zod } from 'sveltekit-superforms/adapters';
import { parse as parseYaml, stringify } from 'yaml';
import { z } from 'zod';
import Component from './activity-options-form.svelte';

//

export const DEFAULT_ACTIVITY_OPTIONS: ActivityOptions = {
	schedule_to_close_timeout: '20m',
	start_to_close_timeout: '20m',
	retry_policy: {
		maximum_attempts: 1
	}
};

export class ActivityOptionsForm {
	#value: ActivityOptions = $state(DEFAULT_ACTIVITY_OPTIONS);
	get value() {
		return this.#value;
	}

	readonly superform: SuperForm<{ code: string }>;

	constructor() {
		this.superform = createForm({
			adapter: zod(
				z.object({
					code: activtyOptionsStringSchema
				})
			),
			initialData: { code: stringify(this.#value) },
			onSubmit: ({ form }) => {
				this.#value = parseYaml(form.data.code);
				this.isOpen = false;
			}
		});

		$effect(() => {
			if (!this.isOpen) return;
			this.superform.form.set({ code: stringify(this.#value) });
		});
	}

	readonly Component = Component;
	isOpen = $state(false);
}

// Schema

const ajv = new Ajv({ allowUnionTypes: true });
const validateActivityOptions = ajv.compile(PipelineSchema.$defs.ActivityOptions);

const activtyOptionsStringSchema = z.string().superRefine((v, ctx) => {
	let res: unknown;
	try {
		res = parseYaml(v);
	} catch (e) {
		ctx.addIssue({
			code: z.ZodIssueCode.custom,
			message: `Invalid YAML document: ${getExceptionMessage(e)}`
		});
		return;
	}

	const isValid = validateActivityOptions(res);
	if (!isValid) {
		const error = ajv.errorsText(validateActivityOptions.errors);
		ctx.addIssue({
			code: z.ZodIssueCode.custom,
			message: `Invalid YAML document: ${error}`
		});
	}
});
