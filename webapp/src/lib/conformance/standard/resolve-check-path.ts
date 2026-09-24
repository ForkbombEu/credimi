// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard, Suite, Version } from '../types';

/** Nest nodes for one filesystem-shaped check path (`std/ver/suite/test`). */
export type ResolvedCheckPath = {
	standard: Standard;
	version: Version;
	suite: Suite;
	test: string;
};

/**
 * Walk a loaded nest tree for a four-segment check id.
 * Returns null when the path shape is wrong or a node is missing.
 */
export function resolveCheckPathFromNest(
	standards: readonly Standard[],
	checkId: string
): ResolvedCheckPath | null {
	const chunks = checkId.split('/');
	if (chunks.length !== 4) return null;

	const [standardUid, versionUid, suiteUid, test] = chunks;
	const standard = standards.find((s) => s.uid === standardUid);
	const version = standard?.versions.find((v) => v.uid === versionUid);
	const suite = version?.suites.find((su) => su.uid === suiteUid);
	if (!standard || !version || !suite) return null;

	return { standard, version, suite, test };
}
