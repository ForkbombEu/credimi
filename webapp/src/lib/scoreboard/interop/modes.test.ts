// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	DEFAULT_INTEROP_MODE,
	INTEROP_MODES,
	isInteropMode,
	normalizeInteropMode
} from './modes';

describe('interop modes', () => {
	it('lists every supported mode once', () => {
		expect(new Set(INTEROP_MODES).size).toBe(INTEROP_MODES.length);
		expect(INTEROP_MODES).toHaveLength(6);
	});

	it('defaults unknown modes to wallets_credentials', () => {
		expect(normalizeInteropMode(null)).toBe(DEFAULT_INTEROP_MODE);
		expect(normalizeInteropMode('bad_mode')).toBe(DEFAULT_INTEROP_MODE);
	});

	it('accepts supported mode strings', () => {
		for (const mode of INTEROP_MODES) {
			expect(isInteropMode(mode)).toBe(true);
			expect(normalizeInteropMode(mode)).toBe(mode);
		}
	});
});
