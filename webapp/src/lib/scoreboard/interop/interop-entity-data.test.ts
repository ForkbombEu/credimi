// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { INTEROP_HUB_COLLECTIONS } from './interop-hub-collections';
import { interopEntityData } from './interop-entity-data';

describe('interop entity data', () => {
	it('covers every hub with a non-empty singular label', () => {
		for (const hub of INTEROP_HUB_COLLECTIONS) {
			const entity = interopEntityData[hub];
			expect(entity).toBeDefined();
			expect(entity.labels.singular.length).toBeGreaterThan(0);
		}
	});
});
