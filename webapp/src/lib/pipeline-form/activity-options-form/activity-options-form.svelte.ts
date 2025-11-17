// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ActivityOptions } from '$pipeline-form/types.generated';
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

type Props = {
	initialData?: ActivityOptions;
};

export const DEFAULT_ACTIVITY_OPTIONS: ActivityOptions = {
	schedule_to_close_timeout: '20m',
	start_to_close_timeout: '20m',
	retry_policy: {
		maximum_attempts: 1
	}
};

export class ActivityOptionsForm {
	constructor(props: Props) {
		this.#value = props.initialData ?? DEFAULT_ACTIVITY_OPTIONS;
	}

	#value: ActivityOptions = $state({});
	get value() {
		return this.#value;
	}

	superform: SuperForm<{ code: string }> | undefined;

	mountForm() {
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
		return this.superform;
	}

	readonly Component = Component;
	isOpen = $state(false);
}

// Schema

const ajv = new Ajv({ allowUnionTypes: true });
export const validateActivityOptions = ajv.compile(PipelineSchema.$defs.ActivityOptions);

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

export function isActivityOptions(value: unknown): value is ActivityOptions {
	return validateActivityOptions(value);
}
