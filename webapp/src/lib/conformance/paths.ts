// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Four path segments: standard UID, version UID, suite UID, test id (pipeline `check_id`). */
export type Path = {
	standard: string;
	version: string;
	suite: string;
	test: string;
};

export function parsePath(path: string): Path {
	const chunks = path.split('/');
	if (chunks.length !== 4) throw new Error('Invalid path');
	return {
		standard: chunks[0],
		version: chunks[1],
		suite: chunks[2],
		test: chunks[3]
	};
}
