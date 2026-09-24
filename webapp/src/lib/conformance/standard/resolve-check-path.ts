// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard, Suite, Version } from '../types';

/**
 * Nest nodes for a filesystem-shaped hub path (`std/ver/suite[/file]`).
 * `file` is set when a fourth path segment is present.
 */
export type ResolvedNestPath = {
	standard: Standard;
	version: Version;
	suite: Suite;
	file?: string;
};

/** Nest nodes for one filesystem-shaped check path (`std/ver/suite/test`). */
export type ResolvedCheckPath = {
	standard: Standard;
	version: Version;
	suite: Suite;
	test: string;
};

/**
 * Walk a loaded nest tree for a suite (3-seg) or check/file (4+-seg) path.
 * Returns null when fewer than 3 segments or a nest node is missing.
 * Matches hub loader shape: optional fourth segment is the file/check id.
 */
export function resolveCheckPathFromNest(
	standards: readonly Standard[],
	path: string
): ResolvedNestPath | null {
	const chunks = path.split('/');
	if (chunks.length < 3) return null;

	const [standardUid, versionUid, suiteUid] = chunks;
	const file = chunks.at(3);

	const standard = standards.find((s) => s.uid === standardUid);
	const version = standard?.versions.find((v) => v.uid === versionUid);
	const suite = version?.suites.find((su) => su.uid === suiteUid);
	if (!standard || !version || !suite) return null;

	if (!file) {
		return { standard, version, suite };
	}
	return { standard, version, suite, file };
}
