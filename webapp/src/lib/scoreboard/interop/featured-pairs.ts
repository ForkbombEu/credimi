// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InteropHubCollection } from './interop-hub-collections';

export type InteropPair = {
	row: InteropHubCollection;
	column: InteropHubCollection;
};

export const FEATURED_INTEROP_PAIRS = [
	{ row: 'wallets', column: 'credentials' },
	{ row: 'wallets', column: 'credential_issuers' },
	{ row: 'wallets', column: 'verifiers' },
	{ row: 'wallets', column: 'use_cases_verifications' },
	{ row: 'wallets', column: 'conformance-checks' },
	{ row: 'use_cases_verifications', column: 'conformance-checks' }
] as const satisfies readonly InteropPair[];

export const DEFAULT_INTEROP_PAIR = FEATURED_INTEROP_PAIRS[0];
