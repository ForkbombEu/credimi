// SPDX-FileCopyrightText: 2026 The Forkbomb Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import typia from 'typia';

type Sample = { id: string };

describe('typia transform setup', () => {
	it('runs the compiler transform for typia.is', () => {
		expect(typia.is<Sample>({ id: 'ok' })).toBe(true);
		expect(typia.is<Sample>({ id: 1 })).toBe(false);
	});
});
