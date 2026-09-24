// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ConformanceSuiteRecord } from './record';

import { HUB_SUITE_SORT_DEFAULT } from './query';
import {
	activeSuiteFacetsFromFilters,
	distinctSuiteFacetValues,
	emptySuiteFacetFilters,
	shouldUseSSRSuiteBrowse
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
