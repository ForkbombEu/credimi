// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { CatalogFacets } from './client';

describe('catalog facet filter options', () => {
	it('accepts protocol sut role provider keys for PB equality filters', () => {
		const facets: CatalogFacets = {
			protocol: 'openid4vp',
			sut: 'wallet_solution',
			role: 'wallet',
			provider: 'ewc'
		};
		expect(Object.keys(facets).sort()).toEqual(['protocol', 'provider', 'role', 'sut']);
	});
});
