// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Array, pipe, Record } from 'effect';

import { parsePath } from './paths';

//

/** One check under a suite row: display id and full path for marketplace links. */
export type GroupedCheckItem = { id: string; path: string };

/** One suite context (standard • version • suite UIDs) with its checks. */
export type GroupedPathsBySuite = {
	title: string;
	standardUid: string;
	versionUid: string;
	suiteUid: string;
	checks: GroupedCheckItem[];
};

/** Group pipeline check paths (`std/ver/suite/test`) by suite row for display. */
export function groupPathsBySuite(paths: string[]): GroupedPathsBySuite[] {
	return pipe(
		paths,
		Array.map(parsePath),
		Array.map((p) => ({
			title: `${p.standard} • ${p.version} • ${p.suite}`,
			standardUid: p.standard,
			versionUid: p.version,
			suiteUid: p.suite,
			checkId: p.test,
			joinedPath: p.joinedPath
		})),
		Array.groupBy((item) => item.title),
		Record.toEntries,
		Array.map(([title, rows]) => {
			const first = rows[0];
			return {
				title,
				standardUid: first.standardUid,
				versionUid: first.versionUid,
				suiteUid: first.suiteUid,
				checks: rows.map((row) => ({ id: row.checkId, path: row.joinedPath }))
			};
		})
	);
}
