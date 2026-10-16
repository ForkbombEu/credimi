// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Pipeline } from '$lib';
import type { Renderable } from '$lib/renderable';
import type { SuperForm } from 'sveltekit-superforms';

import PipelineSchema from '$root/schemas/pipeline/pipeline_schema.json';
import Ajv from 'ajv';
import { zod } from 'sveltekit-superforms/adapters';
import { isMap, parseDocument, parse as parseYaml, stringify } from 'yaml';
import { z } from 'zod/v3';

import { createForm } from '@/forms';
import { getExceptionMessage } from '@/utils/errors';

import Component from './runtime-options-form.svelte';

//

export type RuntimeOptions = NonNullable<Pipeline.Pipeline['runtime']>;

type Props = {
	initialData?: RuntimeOptions;
	isDisabled?: () => boolean;
};

export const DEFAULT_RUNTIME_OPTIONS: RuntimeOptions = {
	disable_android_play_store: false,
	temporal: {
		activity_options: {
			schedule_to_close_timeout: '20m',
			start_to_close_timeout: '20m',
			retry_policy: {
				maximum_attempts: 3
			}
		}
	}
};

export class RuntimeOptionsForm implements Renderable<RuntimeOptionsForm> {
	readonly Component = Component;

	constructor(private props: Props) {
		this.#value = props.initialData ?? DEFAULT_RUNTIME_OPTIONS;
	}

	#value: RuntimeOptions = $state({});
	get value() {
		return this.#value;
	}

	superform: SuperForm<{ code: string }> | undefined;

	mountForm() {
		this.superform = createForm({
			adapter: zod(
				z.object({
					code: runtimeOptionsStringSchema
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

	isOpen = $state(false);

	get disabled() {
		return this.props.isDisabled?.() ?? false;
	}
}

// Schema

const ajv = new Ajv({ allowUnionTypes: true });
export const validateRuntimeOptions = ajv.compile(PipelineSchema.properties.runtime);

const runtimeOptionsStringSchema = z.string().superRefine((v, ctx) => {
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

	const isValid = validateRuntimeOptions(res);
	if (!isValid) {
		const error = ajv.errorsText(validateRuntimeOptions.errors);
		ctx.addIssue({
			code: z.ZodIssueCode.custom,
			message: `Invalid YAML document: ${error}`
		});
	}
});

export function isRuntimeOptions(value: unknown): value is RuntimeOptions {
	return validateRuntimeOptions(value);
}

// no_video toggle: edits the YAML document in place so user comments survive.
// An absent key means false, so turning the flag off removes it.
// Returns undefined when the code is not a YAML mapping (the toggle is then disabled).

function parseRuntimeDocument(code: string) {
	const doc = parseDocument(code);
	if (doc.errors.length > 0) return undefined;
	if (doc.contents !== null && !isMap(doc.contents)) return undefined;
	return doc;
}

export function readNoVideo(code: string): boolean | undefined {
	const doc = parseRuntimeDocument(code);
	if (!doc) return undefined;
	return doc.get('no_video') === true;
}

export function writeNoVideo(code: string, enabled: boolean): string {
	const doc = parseRuntimeDocument(code);
	if (!doc) return code;
	if (enabled) doc.set('no_video', true);
	else if (doc.contents !== null) doc.delete('no_video');
	return doc.toString();
}
