// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Conformance, Hub } from '$lib';
import { entities } from '$lib/global';

import type { Item } from './types';

//

export function fromConformancePaths(paths: string[]): Item[] {
	return Conformance.Check.groupPathsBySuite(paths).map((group) => {
		const suite = Conformance.Standards.resolveSuite(
			group.standardUid,
			group.versionUid,
			group.suiteUid
		);

		return {
			key: `${group.standardUid}/${group.versionUid}/${group.suiteUid}`,
			name: suite.name,
			href: Hub.Conformance.getSuitePageUrl(
				group.standardUid,
				group.versionUid,
				group.suiteUid
			),
			avatar: {
				src: suite.logo,
				fallback: suite.name.slice(0, 2),
				alt: suite.name
			},
			kind: entities.conformance_checks,
			children: group.checks.map((check) => ({
				label: check.id,
				href: Hub.Conformance.getStandardCheckUrlFromPath(check.path)
			}))
		};
	});
}
