// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fg from 'fast-glob';
import fs from 'node:fs';
import path from 'node:path';
import { parse } from 'yaml';

import { formatCode, GENERATED, logCodegenResult } from '@/utils/codegen';

//

type FCAFTestYaml = {
	id?: string;
	title?: string;
	suite?: { section?: string };
	evidence?: Record<string, { from?: unknown }>;
};

type FCAFTestCatalogEntry = {
	id: string;
	title: string;
	section: string;
	sources: string[];
};

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

const testsDir = path.resolve(
	import.meta.dirname,
	'../../../../config_templates/fcaf/wallet_solution/relying_party/tests'
);

const files = await fg(path.join(testsDir, '*.yaml'));

const tests: FCAFTestCatalogEntry[] = [];
for (const file of files) {
	const raw = fs.readFileSync(file, 'utf8');
	const doc = parse(raw) as FCAFTestYaml;
	if (!doc?.id) continue;
	tests.push({
		id: doc.id,
		title: doc.title ?? '',
		section: doc.suite?.section ?? '',
		sources: evidenceSources(doc)
	});
}

tests.sort((a, b) => a.id.localeCompare(b.id));

function evidenceSources(doc: FCAFTestYaml): string[] {
	const sources = new Set<string>();
	for (const binding of Object.values(doc.evidence ?? {})) {
		const from = typeof binding?.from === 'string' ? binding.from : undefined;
		if (!from) continue;
		const marker = '.outputs.';
		const index = from.indexOf(marker);
		if (index > 0) sources.add(from.slice(0, index));
	}
	return [...sources].sort();
}

const pipelinePath = path.resolve(
	import.meta.dirname,
	'../../../../config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml'
);
const pipeline = parse(fs.readFileSync(pipelinePath, 'utf8')) as FCAFPipeline;
const fcafStep = pipeline.steps?.find((step) => step.use === 'fcaf-validation');
const suite = fcafStep?.with?.suite ?? 'wallet_solution/relying_party';
const pipelineOutputs = fcafStep?.with?.pipeline_outputs ?? {};

const code = `
export type FCAFTestCatalogEntry = {
	id: string;
	title: string;
	section: string;
	sources: string[];
};

export const FCAF_TESTS: FCAFTestCatalogEntry[] = ${JSON.stringify(tests, null, 2)};

export const FCAF_SUITE = ${JSON.stringify(suite)};

export const FCAF_PIPELINE_OUTPUTS = ${JSON.stringify(pipelineOutputs, null, 2)};
`;

const formattedCode = await formatCode(code);
const filePath = path.join(import.meta.dirname, `tests.${GENERATED}.ts`);
fs.writeFileSync(filePath, formattedCode);
logCodegenResult('FCAF test catalog', filePath);
