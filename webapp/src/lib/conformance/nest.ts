// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ConformanceCheckRecord } from './record';
import type { Standard, Suite, Version } from './types';

/**
 * Group flat `conformance_checks` rows into the nested standards → versions →
 * suites tree used by start-checks and pipeline pickers.
 *
 * Path identity: each suite `paths[]` entry is the record `path`
 * (`standard/version/suite/stem`). Suite `files[]` keeps the on-disk filename
 * (with extension) for placeholders / start-check submission.
 *
 * Rich standard/version/suite metadata (URLs, logos, descriptions) is not on
 * the v1 catalog row; UIDs are used as display names until enriched later.
 */
export function nestChecks(records: ConformanceCheckRecord[]): Standard[] {
	const byStandard = new Map<string, Map<string, Map<string, ConformanceCheckRecord[]>>>();

	for (const record of records) {
		let byVersion = byStandard.get(record.standard);
		if (!byVersion) {
			byVersion = new Map();
			byStandard.set(record.standard, byVersion);
		}
		let bySuite = byVersion.get(record.version);
		if (!bySuite) {
			bySuite = new Map();
			byVersion.set(record.version, bySuite);
		}
		const checks = bySuite.get(record.suite);
		if (checks) {
			checks.push(record);
		} else {
			bySuite.set(record.suite, [record]);
		}
	}

	const standards: Standard[] = [];

	for (const [standardUid, versionsMap] of byStandard) {
		const versions: Version[] = [];
		for (const [versionUid, suitesMap] of versionsMap) {
			const suites: Suite[] = [];
			for (const [suiteUid, checks] of suitesMap) {
				suites.push({
					uid: suiteUid,
					name: suiteUid,
					homepage: '',
					repository: '',
					help: '',
					description: '',
					files: checks.map((c) => c.file),
					paths: checks.map((c) => c.path)
				});
			}
			versions.push({
				uid: versionUid,
				name: versionUid,
				latest_update: '',
				suites
			});
		}
		standards.push({
			uid: standardUid,
			name: standardUid,
			description: '',
			standard_url: '',
			latest_update: '',
			external_links: null,
			versions
		});
	}

	return standards;
}
