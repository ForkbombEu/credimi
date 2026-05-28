// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from '$lib/conformance/types';

export type ConformanceMetadata = {
	name: string;
	subtitle?: string;
	avatar_url?: string;
};

export function resolveConformanceCheck(
	path: string,
	standards: readonly Standard[]
): ConformanceMetadata | undefined {
	const parts = path.split('/');
	if (parts.length < 4) return undefined;

	const [standardUid, versionUid, suiteUid, ...checkParts] = parts;
	const checkName = checkParts.join('/');

	const standard = standards.find((s) => s.uid === standardUid);
	const version = standard?.versions.find((v) => v.uid === versionUid);
	const suite = version?.suites.find((s) => s.uid === suiteUid);

	if (!suite) return undefined;

	return {
		name: checkName,
		subtitle: suite.name,
		avatar_url: suite.logo
	};
}
