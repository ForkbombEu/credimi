// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Array, pipe, Record } from 'effect';

import { parsePath } from './paths';

//

/** One check under a suite row: display id and full path for marketplace links. */
type GroupedCheckItem = { id: string; path: string };

/** One suite context (standard • version • suite UIDs) with its checks. */
type GroupedPathsBySuite = { title: string; checks: GroupedCheckItem[] };

/** Group pipeline check paths (`std/ver/suite/test`) by suite row for display. */
export function groupPathsBySuite(paths: string[]): GroupedPathsBySuite[] {
	return pipe(
		paths,
		Array.map(parsePath),
		Array.map((p) => ({
			title: `${p.standard} • ${p.version} • ${p.suite}`,
			checkId: p.test,
			joinedPath: p.joinedPath
		})),
		Array.groupBy((item) => item.title),
		Record.toEntries,
		Array.map(([title, rows]) => ({
			title,
			checks: rows.map((row) => ({ id: row.checkId, path: row.joinedPath }))
		}))
	);
}
