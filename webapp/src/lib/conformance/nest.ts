// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ConformanceSuiteRecord } from './record';
import type { Standard, Suite, Version } from './types';

/**
 * Group suite-grain catalog rows into the nested standards → versions → suites
 * tree used by hub, start-checks, and pipeline pickers.
 *
 * Grouping uses filesystem axes (`fs_standard` / `fs_version` / `suite`) so nest
 * URLs stay path-stable (ADR-0002). Suite display metadata comes from the suite
 * row; member checks come from `check_paths` / `check_titles` / `check_files`.
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
				const suiteName = row.suite_name?.trim();
				const suiteHomepage = row.suite_homepage?.trim() ?? '';
				const suiteRepository = row.suite_repository?.trim() ?? '';
				const suiteHelp = row.suite_help?.trim() ?? '';
				const suiteDescription = row.suite_description?.trim() ?? '';
				const suiteLogo = row.suite_logo?.trim() ?? '';

				return {
					uid: row.suite,
					name: suiteName || displayNameFromUid(row.suite),
					homepage: suiteHomepage,
					repository: suiteRepository,
					help: suiteHelp,
					description: suiteDescription,
					...(suiteLogo ? { logo: suiteLogo } : {}),
					files: [...row.check_files],
					paths: [...row.check_paths],
					titles: [...row.check_titles]
				};
			});
			versions.push({
				uid: versionUid,
				name: displayNameFromUid(versionUid),
				latest_update: '',
				suites
			});
		}
		standards.push({
			uid: standardUid,
			name: displayNameFromUid(standardUid),
			description: '',
			standard_url: '',
			latest_update: '',
			external_links: null,
			versions
		});
	}

	return standards;
}

/** Humanize a path UID for display when authored metadata is not on the catalog row. */
export function displayNameFromUid(uid: string): string {
	return uid
		.split(/[_-]+/)
		.filter(Boolean)
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

/** Display labels for normalized product standards. */
const STANDARD_DISPLAY_NAMES: Record<string, string> = {
	openid4vp: 'OpenID4VP',
	openid4vci: 'OpenID4VCI'
};

export function displayStandardName(uid: string): string {
	return STANDARD_DISPLAY_NAMES[uid] ?? displayNameFromUid(uid);
}

/**
 * Resolve a check display title from suite parallel arrays, falling back to the
 * path stem when the path is missing from the suite.
 */
export function titleForCheckPath(
	suite: Pick<Suite, 'paths' | 'titles'>,
	pathOrStem: string
): string {
	const exact = suite.paths.indexOf(pathOrStem);
	if (exact >= 0) {
		const title = suite.titles[exact];
		if (title) return title;
	}
	const bySuffix = suite.paths.findIndex(
		(p) => p.endsWith(`/${pathOrStem}`) || p.split('/').at(-1) === pathOrStem
	);
	if (bySuffix >= 0) {
		const title = suite.titles[bySuffix];
		if (title) return title;
	}
	return pathOrStem.split('/').at(-1) ?? pathOrStem;
}
