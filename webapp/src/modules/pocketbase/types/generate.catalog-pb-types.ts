// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * After pocketbase-typegen, inject ephemeral catalog collections that do not
 * exist in data.db (served by Credimi from :memory:). Idempotent.
 *
 *	bun run src/modules/pocketbase/types/generate.catalog-pb-types.ts
 */
import fs from 'node:fs/promises';
import path from 'node:path';

import { formatCode, GENERATED, logCodegenResult } from '@/utils/codegen';

const MARKER = '/* CREDIMI_EPHEMERAL_CATALOG */';

const RECORD_TYPES = `
${MARKER}
export type ConformanceChecksRecord = {
	id: string
	path: string
	title: string
	standard: string
	version: string
	suite: string
	file: string
	visible_in?: string[]
	protocol?: string
	sut?: string
	role?: string
	provider?: string
	norm_standard?: string
	component?: string
	norm_version?: string
}
export type ConformanceChecksResponse = Required<ConformanceChecksRecord> & BaseSystemFields

export type ConformanceSuitesRecord = {
	id: string
	standard: string
	component?: string
	component_rank?: number
	version?: string
	suite: string
	provider?: string
	suite_name?: string
	suite_homepage?: string
	suite_repository?: string
	suite_help?: string
	suite_description?: string
	suite_logo?: string
	check_count: number
	check_paths?: string[]
	check_titles?: string[]
	check_files?: string[]
	visible_in?: string[]
	fs_standard: string
	fs_version: string
	path_prefix: string
}
export type ConformanceSuitesResponse = Required<ConformanceSuitesRecord> & BaseSystemFields
`;

async function main() {
	const filePath = path.join(import.meta.dirname, `index.${GENERATED}.ts`);
	let source = await fs.readFile(filePath, 'utf8');

	if (source.includes(MARKER)) {
		logCodegenResult('catalog PB types (already injected)', filePath);
		return;
	}

	if (!source.includes('CustomChecks: "custom_checks"')) {
		throw new Error('unexpected Collections const shape in index.generated.ts');
	}
	source = source.replace(
		'CustomChecks: "custom_checks",',
		`CustomChecks: "custom_checks",
	ConformanceChecks: "conformance_checks",
	ConformanceSuites: "conformance_suites",`
	);

	if (!source.includes('// Types containing all Records and Responses')) {
		throw new Error('missing CollectionRecords section marker');
	}
	source = source.replace(
		'// Types containing all Records and Responses, useful for creating typing helper functions',
		`${RECORD_TYPES}

// Types containing all Records and Responses, useful for creating typing helper functions`
	);

	source = source.replace(
		'custom_checks: CustomChecksRecord\n\tfeatures: FeaturesRecord',
		'custom_checks: CustomChecksRecord\n\tconformance_checks: ConformanceChecksRecord\n\tconformance_suites: ConformanceSuitesRecord\n\tfeatures: FeaturesRecord'
	);
	source = source.replace(
		'custom_checks: CustomChecksResponse\n\tfeatures: FeaturesResponse',
		'custom_checks: CustomChecksResponse\n\tconformance_checks: ConformanceChecksResponse\n\tconformance_suites: ConformanceSuitesResponse\n\tfeatures: FeaturesResponse'
	);

	const formatted = await formatCode(source);
	await fs.writeFile(filePath, formatted);
	logCodegenResult('catalog PB types into index.generated', filePath);
}

main().catch((err) => {
	console.error(err);
	process.exit(1);
});
