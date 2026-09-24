// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ConformanceSuiteRecord } from './record';

import { HUB_SUITE_SORT_DEFAULT } from './query';
import {
	activeSuiteFacetsFromFilters,
	distinctSuiteFacetValues,
	dynamicSuiteFacetOptions,
	emptySuiteFacetFilters,
	shouldUseSSRSuiteBrowse,
	suitesMatchingOtherFacets
} from './suite-browse.svelte';

function suite(partial: Partial<ConformanceSuiteRecord>): ConformanceSuiteRecord {
	return {
		id: 'id',
		standard: '',
		component: '',
		component_rank: 9,
		version: '',
		suite: '',
		provider: '',
		provider_label: '',
		suite_name: '',
		suite_subtitle: '',
		suite_homepage: '',
		suite_repository: '',
		suite_help: '',
		suite_description: '',
		suite_logo: '',
		check_count: 0,
		members: [],
		visible_in: [],
		fs_standard: '',
		fs_version: '',
		path_prefix: '',
		...partial
	};
}

describe('shouldUseSSRSuiteBrowse', () => {
	it('is true only for hub-default sort, empty search, no facets, and SSR rows', () => {
		expect(
			shouldUseSSRSuiteBrowse({
				sort: HUB_SUITE_SORT_DEFAULT,
				searchQuery: '',
				hasActiveFilters: false,
				initialSuitesLength: 3
			})
		).toBe(true);
	});

	it('is false when search, facets, non-default sort, or empty SSR payload', () => {
		expect(
			shouldUseSSRSuiteBrowse({
				sort: HUB_SUITE_SORT_DEFAULT,
				searchQuery: 'ewc',
				hasActiveFilters: false,
				initialSuitesLength: 3
			})
		).toBe(false);
		expect(
			shouldUseSSRSuiteBrowse({
				sort: HUB_SUITE_SORT_DEFAULT,
				searchQuery: '',
				hasActiveFilters: true,
				initialSuitesLength: 3
			})
		).toBe(false);
		expect(
			shouldUseSSRSuiteBrowse({
				sort: { kind: 'columns', columns: [{ column: 'suite' }] },
				searchQuery: '',
				hasActiveFilters: false,
				initialSuitesLength: 3
			})
		).toBe(false);
		expect(
			shouldUseSSRSuiteBrowse({
				sort: HUB_SUITE_SORT_DEFAULT,
				searchQuery: '',
				hasActiveFilters: false,
				initialSuitesLength: 0
			})
		).toBe(false);
	});
});

describe('activeSuiteFacetsFromFilters', () => {
	it('keeps only non-empty facet values', () => {
		const filters = emptySuiteFacetFilters();
		filters.standard = 'openid4vp';
		filters.provider = 'ewc';
		expect(activeSuiteFacetsFromFilters(filters)).toEqual({
			standard: 'openid4vp',
			provider: 'ewc'
		});
	});
});

describe('distinctSuiteFacetValues', () => {
	it('returns sorted unique non-empty values for a facet key', () => {
		const records = [
			suite({ standard: 'openid4vp' }),
			suite({ standard: '' }),
			suite({ standard: 'openid4vci' }),
			suite({ standard: 'openid4vp' })
		];
		expect(distinctSuiteFacetValues(records, 'standard')).toEqual([
			'openid4vci',
			'openid4vp'
		]);
	});
});

describe('dynamicSuiteFacetOptions', () => {
	const records = [
		suite({
			id: '1',
			standard: 'openid4vci',
			component: 'wallet',
			provider: 'ewc'
		}),
		suite({
			id: '2',
			standard: 'openid4vci',
			component: 'issuer',
			provider: 'webuild'
		}),
		suite({
			id: '3',
			standard: 'openid4vp',
			component: 'wallet',
			provider: 'ewc'
		}),
		suite({
			id: '4',
			standard: 'openid4vp',
			component: 'verifier',
			provider: 'openid'
		})
	];

	it('lists all values when no filters are active', () => {
		expect(dynamicSuiteFacetOptions(records, emptySuiteFacetFilters())).toEqual({
			standard: ['openid4vci', 'openid4vp'],
			component: ['issuer', 'verifier', 'wallet'],
			provider: ['ewc', 'openid', 'webuild']
		});
	});

	it('narrows other axes from the selected facet, keeping the selected axis open', () => {
		const filters = emptySuiteFacetFilters();
		filters.standard = 'openid4vci';
		expect(dynamicSuiteFacetOptions(records, filters)).toEqual({
			standard: ['openid4vci', 'openid4vp'],
			component: ['issuer', 'wallet'],
			provider: ['ewc', 'webuild']
		});
	});

	it('intersects multiple active facets for remaining axes', () => {
		const filters = emptySuiteFacetFilters();
		filters.standard = 'openid4vp';
		filters.component = 'wallet';
		expect(dynamicSuiteFacetOptions(records, filters)).toEqual({
			standard: ['openid4vci', 'openid4vp'],
			component: ['verifier', 'wallet'],
			provider: ['ewc']
		});
	});
});

describe('suitesMatchingOtherFacets', () => {
	it('ignores the excluded facet when matching', () => {
		const records = [
			suite({ id: 'a', standard: 'openid4vci', provider: 'ewc' }),
			suite({ id: 'b', standard: 'openid4vci', provider: 'webuild' }),
			suite({ id: 'c', standard: 'openid4vp', provider: 'ewc' })
		];
		const filters = emptySuiteFacetFilters();
		filters.standard = 'openid4vci';
		filters.provider = 'ewc';
		expect(suitesMatchingOtherFacets(records, filters, 'provider').map((r) => r.id)).toEqual([
			'a',
			'b'
		]);
	});
});
