// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ConformanceCheckRecord } from './record';
import type { Standard, Suite, Version } from './types';

/**
 * Group flat `conformance_checks` rows into the nested standards → versions →
 * suites tree used by hub, start-checks, and pipeline pickers.
 *
 * Path identity: each suite `paths[]` entry is the record `path`
 * (`standard/version/suite/stem`). Suite `files[]` keeps the on-disk filename
 * (with extension) for placeholders / start-check submission. Suite `titles[]`
 * mirrors those rows with catalog `title` for browse/picker labels.
 *
 * Suite display fields prefer denormalized catalog columns (`suite_name`,
 * `suite_logo`, URLs, description) from suite metadata.yaml (#1399). Standard
 * and version names still use a humanized UID fallback until those axes are
 * denormalized. Empty suites without check files do not appear.
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
				const meta = checks[0];
				const suiteName = meta?.suite_name?.trim();
				const suiteHomepage = meta?.suite_homepage?.trim() ?? '';
				const suiteRepository = meta?.suite_repository?.trim() ?? '';
				const suiteHelp = meta?.suite_help?.trim() ?? '';
				const suiteDescription = meta?.suite_description?.trim() ?? '';
				const suiteLogo = meta?.suite_logo?.trim() ?? '';

				suites.push({
					uid: suiteUid,
					name: suiteName || displayNameFromUid(suiteUid),
					homepage: suiteHomepage,
					repository: suiteRepository,
					help: suiteHelp,
					description: suiteDescription,
					...(suiteLogo ? { logo: suiteLogo } : {}),
					files: checks.map((c) => c.file),
					paths: checks.map((c) => c.path),
					titles: checks.map((c) => c.title)
				});
			}
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
