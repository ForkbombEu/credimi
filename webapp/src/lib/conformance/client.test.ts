// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	appendFacetFilters,
	appendSuiteFacetFilters,
	CATALOG_FACET_KEYS,
	SUITE_FACET_KEYS,
	type CatalogFacets,
	type PbFilterFn,
	type SuiteFacets
} from './client';

/** Deterministic stand-in for `pb.filter` — no live PocketBase needed. */
const stubFilter: PbFilterFn = (raw, params) => {
	let out = raw;
	for (const [key, value] of Object.entries(params ?? {})) {
		out = out.replaceAll(`{:${key}}`, JSON.stringify(value));
	}
	return out;
};

describe('appendFacetFilters', () => {
	it('emits equality filters in protocol → sut → role → provider order', () => {
		const facets: CatalogFacets = {
			protocol: 'openid4vp',
			sut: 'wallet_solution',
			role: 'wallet',
			provider: 'ewc'
		};
		const filters: string[] = [];
		appendFacetFilters(filters, facets, stubFilter);

		expect(CATALOG_FACET_KEYS).toEqual(['protocol', 'sut', 'role', 'provider']);
		expect(filters).toEqual([
			'protocol = "openid4vp"',
			'sut = "wallet_solution"',
			'role = "wallet"',
			'provider = "ewc"'
		]);
	});

	it('omits unset and empty facet values', () => {
		const filters: string[] = [];
		appendFacetFilters(
			filters,
			{ protocol: 'openid4vp', sut: '', role: undefined, provider: 'ewc' },
			stubFilter
		);

		expect(filters).toEqual(['protocol = "openid4vp"', 'provider = "ewc"']);
	});

	it('is a no-op when facets are undefined', () => {
		const filters: string[] = ['standard = "fcaf"'];
		appendFacetFilters(filters, undefined, stubFilter);
		expect(filters).toEqual(['standard = "fcaf"']);
	});

	it('composes with preceding surface/standard filters via && join', () => {
		const filters: string[] = [
			stubFilter('visible_in ~ {:surface}', { surface: 'hub' }),
			stubFilter('standard = {:standard}', { standard: 'fcaf' })
		];
		appendFacetFilters(filters, { role: 'wallet', protocol: 'openid4vp' }, stubFilter);

		expect(filters.join(' && ')).toBe(
			'visible_in ~ "hub" && standard = "fcaf" && protocol = "openid4vp" && role = "wallet"'
		);
	});

	it('passes field-keyed params to the filter function', () => {
		const calls: Array<{ raw: string; params?: Record<string, unknown> }> = [];
		const recordingFilter: PbFilterFn = (raw, params) => {
			calls.push({ raw, params });
			return stubFilter(raw, params);
		};

		appendFacetFilters(
			[],
			{ sut: 'wallet_solution', provider: 'ewc' },
			recordingFilter
		);

		expect(calls).toEqual([
			{ raw: 'sut = {:sut}', params: { sut: 'wallet_solution' } },
			{ raw: 'provider = {:provider}', params: { provider: 'ewc' } }
		]);
	});
});

describe('appendSuiteFacetFilters', () => {
	it('emits equality filters in standard → component → version → provider order', () => {
		const facets: SuiteFacets = {
			standard: 'openid4vp',
			component: 'wallet',
			version: 'draft-24',
			provider: 'openid_conformance_suite'
		};
		const filters: string[] = [];
		appendSuiteFacetFilters(filters, facets, stubFilter);

		expect(SUITE_FACET_KEYS).toEqual(['standard', 'component', 'version', 'provider']);
		expect(filters).toEqual([
			'standard = "openid4vp"',
			'component = "wallet"',
			'version = "draft-24"',
			'provider = "openid_conformance_suite"'
		]);
	});

	it('omits empty suite facet values', () => {
		const filters: string[] = [];
		appendSuiteFacetFilters(
			filters,
			{ standard: 'openid4vci', component: '', version: undefined, provider: 'webuild' },
			stubFilter
		);
		expect(filters).toEqual(['standard = "openid4vci"', 'provider = "webuild"']);
	});
});
