// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	compileCheckListQuery,
	compileSuiteListQuery,
	HUB_SUITE_SORT_DEFAULT,
	isHubDefaultSuiteSort,
	SUITE_FACET_KEYS,
	suiteSortFromTableColumns,
	type FilterCompiler
} from './query';

/** Deterministic stand-in for `pb.filter` — no live PocketBase needed. */
const stubFilter: FilterCompiler = (raw, params) => {
	let out = raw;
	for (const [key, value] of Object.entries(params ?? {})) {
		out = out.replaceAll(`{:${key}}`, JSON.stringify(value));
	}
	return out;
};

describe('compileSuiteListQuery', () => {
	it('uses hub default sort with no filter when intent is empty', () => {
		expect(compileSuiteListQuery({}, stubFilter)).toEqual({
			sort: 'component_rank,standard,suite'
		});
	});

	it('compiles surface, product facets, and search into one filter', () => {
		const compiled = compileSuiteListQuery(
			{
				surface: 'pipeline',
				facets: {
					standard: 'openid4vp',
					component: 'wallet',
					version: 'draft-24',
					provider: 'openid_conformance_suite'
				},
				search: '  ewc  '
			},
			stubFilter
		);

		expect(SUITE_FACET_KEYS).toEqual(['standard', 'component', 'version', 'provider']);
		expect(compiled.sort).toBe('component_rank,standard,suite');
		expect(compiled.filter).toBe(
			[
				'visible_in ~ "pipeline"',
				'standard = "openid4vp"',
				'component = "wallet"',
				'version = "draft-24"',
				'provider = "openid_conformance_suite"',
				'(suite_name ~ "ewc" || suite ~ "ewc" || standard ~ "ewc" || component ~ "ewc" || version ~ "ewc" || provider ~ "ewc")'
			].join(' && ')
		);
	});

	it('omits empty facet values and whitespace-only search', () => {
		const compiled = compileSuiteListQuery(
			{
				facets: {
					standard: 'openid4vci',
					component: '',
					version: undefined,
					provider: 'webuild'
				},
				search: '   '
			},
			stubFilter
		);

		expect(compiled.filter).toBe('standard = "openid4vci" && provider = "webuild"');
	});

	it('maps domain sort columns to PocketBase fields (suite → suite_name)', () => {
		const compiled = compileSuiteListQuery(
			{
				sort: {
					kind: 'columns',
					columns: [
						{ column: 'component', desc: true },
						{ column: 'standard' },
						{ column: 'suite', desc: true }
					]
				}
			},
			stubFilter
		);

		expect(compiled.sort).toBe('-component_rank,standard,-suite_name');
	});
});

describe('compileCheckListQuery', () => {
	it('compiles surface and FS standard with default check sort', () => {
		const compiled = compileCheckListQuery(
			{ surface: 'manual', fs_standard: 'fcaf' },
			stubFilter
		);

		expect(compiled.sort).toBe('fs_standard,fs_version,suite,path');
		expect(compiled.filter).toBe('visible_in ~ "manual" && fs_standard = "fcaf"');
	});

	it('applies default check sort with no filter when intent is empty', () => {
		expect(compileCheckListQuery({}, stubFilter)).toEqual({
			sort: 'fs_standard,fs_version,suite,path'
		});
	});

	it('passes field-keyed params to the injected filter compiler', () => {
		const calls: Array<{ raw: string; params?: Record<string, unknown> }> = [];
		const recordingFilter: FilterCompiler = (raw, params) => {
			calls.push({ raw, params });
			return stubFilter(raw, params);
		};

		compileCheckListQuery({ surface: 'pipeline', fs_standard: 'fcaf' }, recordingFilter);

		expect(calls).toEqual([
			{ raw: 'visible_in ~ {:surface}', params: { surface: 'pipeline' } },
			{ raw: 'fs_standard = {:fs_standard}', params: { fs_standard: 'fcaf' } }
		]);
	});
});

describe('suiteSortFromTableColumns', () => {
	it('returns hub default for empty or unknown columns', () => {
		expect(suiteSortFromTableColumns([])).toEqual(HUB_SUITE_SORT_DEFAULT);
		expect(suiteSortFromTableColumns([{ id: 'version', desc: false }])).toEqual(
			HUB_SUITE_SORT_DEFAULT
		);
		expect(isHubDefaultSuiteSort(undefined)).toBe(true);
		expect(isHubDefaultSuiteSort(HUB_SUITE_SORT_DEFAULT)).toBe(true);
	});

	it('keeps only known suite sort columns', () => {
		expect(
			suiteSortFromTableColumns([
				{ id: 'suite', desc: true },
				{ id: 'noise', desc: false },
				{ id: 'standard', desc: false }
			])
		).toEqual({
			kind: 'columns',
			columns: [{ column: 'suite', desc: true }, { column: 'standard' }]
		});
	});
});
