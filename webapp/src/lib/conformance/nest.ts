// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ConformanceSuiteRecord } from './record';
import {
	displayNameFromUid,
	suiteLogo,
	suiteSubtitle,
	suiteTitle
} from './suite-present';
import type { Standard, Suite, Version } from './types';

/**
 * Group suite-grain catalog rows into the nested standards → versions → suites
 * tree used by hub detail, start-checks, and pipeline pickers.
 *
 * Dual browse axes (intentional): the hub suite **table** sorts/filters on product
 * fields (`standard` / `component` / `version`); this nest groups on filesystem
 * axes (`fs_standard` / `fs_version` / `suite`) so `/hub/conformance-checks/{path}`
 * stays path-stable (ADR-0002). Nest standard/version nodes are path-axis labels
 * (uid + display name), not authored standard.yaml / version.yaml metadata.
 * Load nest only where detail/pickers need it — not on the hub table route.
 * Suite display metadata and member checks come from the suite row (`members`).
 * Empty suites without checks do not appear in the catalog projection.
 */
export function nestSuites(records: ConformanceSuiteRecord[]): Standard[] {
	const byStandard = new Map<string, Map<string, ConformanceSuiteRecord[]>>();

	for (const record of records) {
		const fsStandard = record.fs_standard;
		const fsVersion = record.fs_version;
		let byVersion = byStandard.get(fsStandard);
		if (!byVersion) {
			byVersion = new Map();
			byStandard.set(fsStandard, byVersion);
		}
		const suites = byVersion.get(fsVersion);
		if (suites) {
			suites.push(record);
		} else {
			byVersion.set(fsVersion, [record]);
		}
	}

	const standards: Standard[] = [];

	for (const [standardUid, versionsMap] of byStandard) {
		const versions: Version[] = [];
		for (const [versionUid, suiteRows] of versionsMap) {
			const suites: Suite[] = suiteRows.map((row) => {
				const subtitle = suiteSubtitle(row);
				const logo = suiteLogo(row);
				const suiteHomepage = row.suite_homepage?.trim() ?? '';
				const suiteRepository = row.suite_repository?.trim() ?? '';
				const suiteHelp = row.suite_help?.trim() ?? '';
				const suiteDescription = row.suite_description?.trim() ?? '';

				return {
					uid: row.suite,
					name: suiteTitle(row),
					...(subtitle ? { subtitle } : {}),
					homepage: suiteHomepage,
					repository: suiteRepository,
					help: suiteHelp,
					description: suiteDescription,
					...(logo ? { logo } : {}),
					members: row.members.map((m) => ({ ...m }))
				};
			});
			versions.push({
				uid: versionUid,
				name: displayNameFromUid(versionUid),
				suites
			});
		}
		standards.push({
			uid: standardUid,
			name: displayNameFromUid(standardUid),
			versions
		});
	}

	return standards;
}

/**
 * Resolve a check display title from suite members, falling back to the path
 * stem when the path is missing from the suite.
 */
export function titleForCheckPath(
	suite: Pick<Suite, 'members'>,
	pathOrStem: string
): string {
	const exact = suite.members.find((m) => m.path === pathOrStem);
	if (exact?.title) return exact.title;
	const bySuffix = suite.members.find(
		(m) => m.path.endsWith(`/${pathOrStem}`) || m.path.split('/').at(-1) === pathOrStem
	);
	if (bySuffix?.title) return bySuffix.title;
	return pathOrStem.split('/').at(-1) ?? pathOrStem;
}
