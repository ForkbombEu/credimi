// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Array, pipe, Record } from 'effect';

//

export function groupPathsByStandard(paths: string[]) {
	return pipe(
		paths,
		Array.map((string) => {
			const [standard, version, suite, test] = string.split('/');
			return {
				title: `${standard} • ${version} • ${suite}`,
				test
			};
		}),
		Array.groupBy((x) => x.title),
		Record.toEntries,
		Array.map(([k, v]) => ({
			title: k,
			items: v.map((x) => x.test)
		}))
	);
}
