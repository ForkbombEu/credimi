// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Generates FCAF validation step defaults (default suite + per-suite
 * pipeline_outputs) from each suite's aggregate complete-validation pipeline.
 * Test listing comes from the shared PocketBase conformance catalog — do not
 * re-emit FCAF_TESTS here.
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

/** Suite of a new fcaf-validation step. */
const DEFAULT_SUITE = 'wallet_solution/relying_party';

const templatesRoot = path.resolve(import.meta.dirname, '../../../../config_templates/fcaf');

/** Every `<sut>/<role>/pipelines/fcaf-*-complete-validation.yaml` under the templates root. */
function completeValidationPipelines(): string[] {
	const directories = (dir: string) =>
		fs
			.readdirSync(dir, { withFileTypes: true })
			.filter((entry) => entry.isDirectory())
			.map((entry) => path.join(dir, entry.name));
	return directories(templatesRoot)
		.flatMap(directories)
		.map((suiteDir) => path.join(suiteDir, 'pipelines'))
		.filter((pipelinesDir) => fs.existsSync(pipelinesDir))
		.flatMap((pipelinesDir) =>
			fs
				.readdirSync(pipelinesDir)
				.filter((name) => /^fcaf-.*-complete-validation\.yaml$/.test(name))
				.map((name) => path.join(pipelinesDir, name))
		)
		.sort();
}

const pipelineOutputsBySuite: Record<string, Record<string, unknown>> = {};
for (const pipelinePath of completeValidationPipelines()) {
	const pipeline = parse(fs.readFileSync(pipelinePath, 'utf8')) as FCAFPipeline;
	const fcafStep = pipeline.steps?.find((step) => step.use === 'fcaf-validation');
	const suite = fcafStep?.with?.suite;
	if (!suite) throw new Error(`${pipelinePath} has no fcaf-validation step with a suite`);
	pipelineOutputsBySuite[suite] = fcafStep?.with?.pipeline_outputs ?? {};
}
if (!pipelineOutputsBySuite[DEFAULT_SUITE]) {
	throw new Error(`no complete-validation pipeline found for ${DEFAULT_SUITE}`);
}

const code = `
/** Suite path default for new fcaf-validation steps. */
export const FCAF_SUITE = ${JSON.stringify(DEFAULT_SUITE)};

/**
 * Full pipeline_outputs map of each suite's aggregate FCAF complete-validation
 * pipeline, keyed by suite. Used as step defaults when any test ids are selected.
 */
export const FCAF_PIPELINE_OUTPUTS_BY_SUITE: Record<string, Record<string, unknown>> = ${JSON.stringify(pipelineOutputsBySuite, null, 2)};
`;

const formattedCode = await formatCode(code);
const filePath = path.join(import.meta.dirname, `tests.${GENERATED}.ts`);
fs.writeFileSync(filePath, formattedCode);
logCodegenResult('FCAF validation defaults', filePath);
