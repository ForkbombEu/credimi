// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export function getLastPathSegment(path: string) {
	return path.split('/').filter(Boolean).at(-1) ?? '';
}
