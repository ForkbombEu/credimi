// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { StandardsWithTestSuites } from './standards-response-schema';

export class SelectStandardAndSuitesForm {
	constructor(private data: StandardsWithTestSuites) {}
}
