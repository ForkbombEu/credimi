// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Generates FCAF validation step defaults (suite + pipeline_outputs) from the
 * aggregate complete-validation pipeline. Test listing comes from the shared
 * PocketBase conformance catalog — do not re-emit FCAF_TESTS here.
 */

import fs from 'node:fs';
import path from 'node:path';
import { parse } from 'yaml';

import { formatCode, GENERATED, logCodegenResult } from '@/utils/codegen';

//

type FCAFValidationStep = {
	use?: string;
	with?: {
		suite?: string;
		pipeline_outputs?: Record<string, unknown>;
	};
};

type FCAFPipeline = {
	steps?: FCAFValidationStep[];
};

const pipelinePath = path.resolve(
	import.meta.dirname,
	'../../../../config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml'
);
const pipeline = parse(fs.readFileSync(pipelinePath, 'utf8')) as FCAFPipeline;
const fcafStep = pipeline.steps?.find((step) => step.use === 'fcaf-validation');
const suite = fcafStep?.with?.suite ?? 'wallet_solution/relying_party';
const pipelineOutputs = fcafStep?.with?.pipeline_outputs ?? {};

const code = `
/** Suite path default for new fcaf-validation steps. */
export const FCAF_SUITE = ${JSON.stringify(suite)};

/**
 * Full pipeline_outputs map from the aggregate FCAF complete-validation pipeline.
 * Used as step defaults when any test ids are selected.
 */
export const FCAF_PIPELINE_OUTPUTS = ${JSON.stringify(pipelineOutputs, null, 2)};
`;

const formattedCode = await formatCode(code);
const filePath = path.join(import.meta.dirname, `tests.${GENERATED}.ts`);
fs.writeFileSync(filePath, formattedCode);
logCodegenResult('FCAF validation defaults', filePath);
