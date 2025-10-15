// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import stepciJsonSchema from '$root/schemas/stepci/schema.json';
import Ajv from 'ajv';
import { Record as R } from 'effect';
import { isCollection, parseAllDocuments, parse as parseYaml } from 'yaml';
import { z } from 'zod';

import { getExceptionMessage } from '@/utils/errors';

//

export const yamlStringSchema = z
	.string()
	.nonempty()
	.superRefine((value, ctx) => {
		const docs = parseAllDocuments(value);

		// @ts-expect-error - `docs.empty` may exist but is not typed
		if (docs.empty) {
			ctx.addIssue({
				code: z.ZodIssueCode.custom,
				message: 'Empty YAML document'
			});
			return;
		}

		for (const [index, doc] of Object.entries(docs)) {
			const i = parseInt(index) + 1;
			const prefix = docs.length > 1 ? `Document ${i}: ` : '';
			if (doc.errors.length > 0) {
				ctx.addIssue({
					code: z.ZodIssueCode.custom,
					message: `${prefix}${doc.errors.join(' | ')}`
				});
			} else if (!isCollection(doc.contents)) {
				ctx.addIssue({
					code: z.ZodIssueCode.custom,
					message: `${prefix}Not a JSON object`
				});
			}
		}
	});

export const jsonStringSchema = z.string().superRefine((v, ctx) => {
	try {
		if (v.length === 0) {
			return {};
		} else {
			z.record(z.string(), z.unknown())
				.refine((value) => R.size(value) > 0)
				.parse(JSON.parse(v));
		}
	} catch (e) {
		const message = getExceptionMessage(e);
		ctx.addIssue({
			code: z.ZodIssueCode.custom,
			message: `Invalid JSON object: ${message}`
		});
	}
});

//

const ajv = new Ajv({ allowUnionTypes: true });
const validateStepci = ajv.compile(stepciJsonSchema);

export const stepciYamlSchema = z.string().superRefine((v, ctx) => {
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

	const isValid = validateStepci(res);
	if (!isValid) {
		const error = ajv.errorsText(validateStepci.errors);
		ctx.addIssue({
			code: z.ZodIssueCode.custom,
			message: `Invalid YAML document: ${error}`
		});
	}
});
