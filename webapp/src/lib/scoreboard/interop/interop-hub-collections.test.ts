// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	INTEROP_HUB_COLLECTIONS,
	interopHubEntity,
	isInteropHubCollection
} from './interop-hub-collections';

describe('interop hub collections', () => {
	it('lists six unique hub collections', () => {
		expect(new Set(INTEROP_HUB_COLLECTIONS).size).toBe(6);
		expect(INTEROP_HUB_COLLECTIONS).toHaveLength(6);
	});

	it('guards known hubs', () => {
		expect(isInteropHubCollection('wallets')).toBe(true);
		expect(isInteropHubCollection('conformance-checks')).toBe(true);
		expect(isInteropHubCollection('bad')).toBe(false);
	});
});

describe('interopHubEntity', () => {
	it('covers every hub with a non-empty singular label', () => {
		for (const hub of INTEROP_HUB_COLLECTIONS) {
			const entity = interopHubEntity(hub);
			expect(entity.labels.singular.length).toBeGreaterThan(0);
		}
	});
});
