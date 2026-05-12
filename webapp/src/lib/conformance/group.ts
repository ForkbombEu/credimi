// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Array, pipe, Record } from 'effect';

import { parsePath } from './paths';

//

/** One suite context (standard • version • suite UIDs) with its check ids. */
export type GroupedPathsBySuite = { title: string; items: string[] };

/** Group pipeline check paths (`std/ver/suite/test`) by suite row for display. */
export function groupPathsBySuite(paths: string[]): GroupedPathsBySuite[] {
	return pipe(
		paths,
		Array.map(parsePath),
		Array.map((p) => ({
			title: `${p.standard} • ${p.version} • ${p.suite}`,
			test: p.test
		})),
		Array.groupBy((x) => x.title),
		Record.toEntries,
		Array.map(([title, rows]) => ({
			title,
			items: rows.map((x) => x.test)
		}))
	);
}
