// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Parsed pipeline check path: four Filesystem-axis segments plus canonical joined form. */
export type Path = {
	fsStandard: string;
	fsVersion: string;
	suite: string;
	test: string;
	/** Same segments joined as `std/ver/suite/test` (canonical for URLs and APIs). */
	joinedPath: string;
};

export function parsePath(path: string): Path {
	const chunks = path.split('/');
	if (chunks.length !== 4) throw new Error('Invalid path');
	const [fsStandard, fsVersion, suite, test] = chunks;
	const joinedPath = `${fsStandard}/${fsVersion}/${suite}/${test}`;
	return {
		fsStandard,
		fsVersion,
		suite,
		test,
		joinedPath
	};
}
