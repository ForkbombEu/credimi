// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import PipelineSchema from '$root/schemas/pipeline/pipeline_schema.json';
import Ajv from 'ajv/dist/2020';
import { parse as parseYaml } from 'yaml';

import { getExceptionMessage } from '@/utils/errors';

export type PipelineYamlValidation =
	| { ok: true; value: string }
	| { ok: false; message: string };

const ajv = new Ajv({ allowUnionTypes: true, dynamicRef: true });
const validatePipeline = ajv.compile(PipelineSchema);

export function validateYaml(yaml: string): PipelineYamlValidation {
	let parsed: unknown;
	try {
		parsed = parseYaml(yaml);
	} catch (e) {
		return { ok: false, message: `Invalid YAML document: ${getExceptionMessage(e)}` };
	}

	if (!validatePipeline(parsed)) {
		return { ok: false, message: `Invalid YAML document: ${ajv.errorsText(validatePipeline.errors)}` };
	}

	return { ok: true, value: yaml };
}
