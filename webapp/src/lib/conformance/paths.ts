// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Parsed pipeline check path: four segments plus canonical joined form. */
export type Path = {
	standard: string;
	version: string;
	suite: string;
	test: string;
	/** Same segments joined as `std/ver/suite/test` (canonical for URLs and APIs). */
	joinedPath: string;
};

export function parsePath(path: string): Path {
	const chunks = path.split('/');
	if (chunks.length !== 4) throw new Error('Invalid path');
	const [standard, version, suite, test] = chunks;
	const joinedPath = `${standard}/${version}/${suite}/${test}`;
	return {
		standard,
		version,
		suite,
		test,
		joinedPath
	};
}
