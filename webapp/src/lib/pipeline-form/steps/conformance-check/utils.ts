// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getLastPathSegment } from '../_partials/misc';

export function getTestName(test: string): string {
	return getLastPathSegment(test);
}
