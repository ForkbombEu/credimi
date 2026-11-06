// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { PendingReview } from './api';

import { canAccept } from './review';

// sha256("a")
const SHA_A = 'ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb';

function review(overrides: Partial<PendingReview>): PendingReview {
	return {
		tool_id: 't1',
		name: 'install',
		build: 'org/w',
		code: 'a',
		candidate_sha256: SHA_A,
		plan: { steps: [] },
		replaces: '',
		error: '',
		...overrides
	};
}

describe('canAccept', () => {
	it('refuses a wrong hash', async () => {
		expect(await canAccept(review({ candidate_sha256: '0'.repeat(64) }))).toBe(false);
	});

	it('refuses a candidate without a plan', async () => {
		expect(await canAccept(review({ plan: null }))).toBe(false);
	});

	it('accepts the matching YAML with a plan', async () => {
		expect(await canAccept(review({}))).toBe(true);
	});
});
