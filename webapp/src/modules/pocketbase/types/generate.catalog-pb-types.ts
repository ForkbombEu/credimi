// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * After pocketbase-typegen, inject ephemeral catalog collections that do not
 * exist in data.db (served by Credimi from :memory:). Idempotent.
 *
 * Record type bodies come from catalog-pb-records.ts
 * (`go generate ./pkg/conformancecatalog`).
 *
 *	bun run src/modules/pocketbase/types/generate.catalog-pb-types.ts
 */
import fs from 'node:fs/promises';
import path from 'node:path';

import { formatCode, GENERATED, logCodegenResult } from '@/utils/codegen';

const MARKER = '/* CREDIMI_EPHEMERAL_CATALOG */';

async function main() {
	const dir = import.meta.dirname;
	const filePath = path.join(dir, `index.${GENERATED}.ts`);
	const recordsPath = path.join(dir, 'catalog-pb-records.ts');

	const recordsSource = await fs.readFile(recordsPath, 'utf8');
	const recordTypes = extractRecordTypes(recordsSource);

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
		`${MARKER}
${recordTypes}
export type ConformanceChecksResponse = Required<ConformanceChecksRecord> & BaseSystemFields
export type ConformanceSuitesResponse = Required<ConformanceSuitesRecord> & BaseSystemFields

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

/** Strip SPDX/header comments; keep export type … bodies for injection. */
function extractRecordTypes(source: string): string {
	const start = source.indexOf('export type ConformanceChecksRecord');
	if (start < 0) {
		throw new Error('catalog-pb-records.ts missing ConformanceChecksRecord');
	}
	return source.slice(start).trimEnd();
}

main().catch((err) => {
	console.error(err);
	process.exit(1);
});
