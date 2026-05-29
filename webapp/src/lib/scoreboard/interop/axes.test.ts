// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { INTEROP_AXIS_KEYS, interopAxisLabel, isInteropAxisKey } from './axes';

describe('interop axes', () => {
	it('lists every supported axis key once', () => {
		expect(new Set(INTEROP_AXIS_KEYS).size).toBe(INTEROP_AXIS_KEYS.length);
		expect(INTEROP_AXIS_KEYS).toHaveLength(6);
	});

	it('labels known axis keys', () => {
		for (const key of INTEROP_AXIS_KEYS) {
			expect(isInteropAxisKey(key)).toBe(true);
			expect(interopAxisLabel(key)).not.toBe(key);
			expect(interopAxisLabel(key).length).toBeGreaterThan(0);
		}
	});

	it('returns unknown keys unchanged', () => {
		expect(interopAxisLabel('future_axis')).toBe('future_axis');
	});
});
