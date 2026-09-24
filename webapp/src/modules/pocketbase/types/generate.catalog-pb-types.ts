// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * After pocketbase-typegen, inject ephemeral catalog collections that do not
 * exist in data.db (served by Credimi from :memory:). Stitch-only: Record type
 * bodies come from catalog-pb-records.ts (`go generate ./pkg/conformancecatalog`).
 * When the inject marker is already present, Record bodies are replaced so a
 * Client column change does not require a typegen wipe.
 *
 * Matching tolerates pocketbase-typegen double quotes and Prettier single quotes
 * (formatCode runs after each write).
 *
 *	bun run src/modules/pocketbase/types/generate.catalog-pb-types.ts
 */
import fs from 'node:fs/promises';
import path from 'node:path';

import { formatCode, GENERATED, logCodegenResult } from '@/utils/codegen';

const MARKER = '/* CREDIMI_EPHEMERAL_CATALOG */';
const RECORDS_SECTION =
	'// Types containing all Records and Responses, useful for creating typing helper functions';

async function main() {
	const dir = import.meta.dirname;
	const filePath = path.join(dir, `index.${GENERATED}.ts`);
	const recordsPath = path.join(dir, 'catalog-pb-records.ts');

	const recordsSource = await fs.readFile(recordsPath, 'utf8');
	const recordTypes = extractRecordTypes(recordsSource);

	let source = await fs.readFile(filePath, 'utf8');

	if (!/ConformanceChecks:\s*['"]conformance_checks['"]/.test(source)) {
		if (!/CustomChecks:\s*['"]custom_checks['"]/.test(source)) {
			throw new Error('unexpected Collections const shape in index.generated.ts');
		}
		source = source.replace(
			/CustomChecks:\s*['"]custom_checks['"],/,
			`CustomChecks: "custom_checks",
	ConformanceChecks: "conformance_checks",
	ConformanceSuites: "conformance_suites",`
		);
	}

	if (!/conformance_checks:\s*ConformanceChecksRecord/.test(source)) {
		source = source.replace(
			/custom_checks:\s*CustomChecksRecord;?\n\tfeatures:\s*FeaturesRecord/,
			'custom_checks: CustomChecksRecord\n\tconformance_checks: ConformanceChecksRecord\n\tconformance_suites: ConformanceSuitesRecord\n\tfeatures: FeaturesRecord'
		);
	}
	if (!/conformance_checks:\s*ConformanceChecksResponse/.test(source)) {
		source = source.replace(
			/custom_checks:\s*CustomChecksResponse;?\n\tfeatures:\s*FeaturesResponse/,
			'custom_checks: CustomChecksResponse\n\tconformance_checks: ConformanceChecksResponse\n\tconformance_suites: ConformanceSuitesResponse\n\tfeatures: FeaturesResponse'
		);
	}

	const injectBlock = `${MARKER}
${recordTypes}
export type ConformanceChecksResponse = Required<ConformanceChecksRecord> & BaseSystemFields
export type ConformanceSuitesResponse = Required<ConformanceSuitesRecord> & BaseSystemFields

`;

	if (source.includes(MARKER)) {
		const start = source.indexOf(MARKER);
		const end = source.indexOf(RECORDS_SECTION, start);
		if (end < 0) {
			throw new Error('catalog inject block missing Records section marker');
		}
		source = source.slice(0, start) + injectBlock + source.slice(end);
		logCodegenResult('catalog PB types (upserted Record bodies)', filePath);
	} else {
		if (!source.includes(RECORDS_SECTION)) {
			throw new Error('missing CollectionRecords section marker');
		}
		source = source.replace(RECORDS_SECTION, `${injectBlock}${RECORDS_SECTION}`);
		logCodegenResult('catalog PB types into index.generated', filePath);
	}

	const formatted = await formatCode(source);
	await fs.writeFile(filePath, formatted);
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
