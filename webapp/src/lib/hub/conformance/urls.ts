// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Base path for public conformance hub routes. */
export const hubConformanceChecksPath = '/hub/conformance-checks';

/**
 * URL for a conformance suite collection page (standard + version + suite UIDs).
 * Matches `webapp/src/routes/(public)/hub/conformance-checks/[...path]/+page.ts` (minimum path depth).
 */
export function getSuitePageUrl(standardUid: string, versionUid: string, suiteUid: string): string {
	return `${hubConformanceChecksPath}/${standardUid}/${versionUid}/${suiteUid}`;
}

/**
 * URL for a single conformance check (suite file / pipeline `check_id` tail).
 * `checkId` is the final path segment (no slashes), e.g. YAML stem used in routes.
 */
export function getStandardCheckUrl(
	standardUid: string,
	versionUid: string,
	suiteUid: string,
	checkId: string
): string {
	return `${getSuitePageUrl(standardUid, versionUid, suiteUid)}/${checkId}`;
}

/** Joined `standard/version/suite/test` path as returned by suite.paths / pipeline `check_id`. */
export function getStandardCheckUrlFromPath(path: string): string {
	return `${hubConformanceChecksPath}/${path}`;
}
