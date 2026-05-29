// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { Standard } from '$lib/conformance/types';

import { m } from '@/i18n';

import type { InteropMatrixResponse } from './types';

import { hubHref, subtitleOrVersion, toViewMatrix } from './to-view-matrix';

const standards: Standard[] = [
	{
		uid: 'std',
		name: 'Standard',
		description: '',
		standard_url: '',
		latest_update: '',
		external_links: null,
		versions: [
			{
				uid: 'ver',
				name: '1.0',
				latest_update: '',
				suites: [
					{
						uid: 'suite',
						name: 'Suite Name',
						homepage: '',
						repository: '',
						help: '',
						description: '',
						logo: 'https://example.com/logo.png',
						files: [],
						paths: []
					}
				]
			}
		]
	}
];

function minimalMatrix(overrides: Partial<InteropMatrixResponse> = {}): InteropMatrixResponse {
	return {
		row: { hub_collection: 'wallets', path_based: false },
		column: { hub_collection: 'credential_issuers', path_based: false },
		rows: [
			{
				id: 'w1',
				name: 'Wallet One',
				path: 'org/w1',
				version_label: 'v2'
			}
		],
		columns: [
			{
				id: 'i1',
				name: 'Issuer One',
				path: 'org/i1',
				subtitle: 'Issuer sub'
			}
		],
		cells: [
			{
				row_id: 'w1',
				column_id: 'i1',
				pipeline_count: 1,
				total_runs: 10,
				total_successes: 9,
				success_rate: 90,
				status: 'stable'
			}
		],
		...overrides
	};
}

const viewOptions = { standards };

describe('hubHref', () => {
	it('joins hub collection and entity path', () => {
		expect(hubHref({ hub_collection: 'wallets', path_based: false }, 'org/w1')).toBe(
			'/hub/wallets/org/w1'
		);
	});

	it('uses conformance-checks segment for path-based columns', () => {
		expect(
			hubHref(
				{ hub_collection: 'conformance-checks', path_based: true },
				'std/ver/suite/check-a'
			)
		).toBe('/hub/conformance-checks/std/ver/suite/check-a');
	});
});

describe('subtitleOrVersion', () => {
	it('prefers subtitle over version label', () => {
		expect(subtitleOrVersion('sub', 'v1')).toBe('sub');
	});

	it('falls back to version label', () => {
		expect(subtitleOrVersion(undefined, 'v1')).toBe('v1');
	});

	it('returns undefined when both are missing', () => {
		expect(subtitleOrVersion(undefined, undefined)).toBeUndefined();
	});
});

describe('toViewMatrix', () => {
	it('builds corner label, entity hrefs, subtitles, and cell lookup', () => {
		const view = toViewMatrix(minimalMatrix(), viewOptions);

		expect(view.cornerLabel).toBe(
			m.interop_matrix_corner_label({ row: m.Wallet(), column: m.Issuer() })
		);
		expect(view.rows).toEqual([
			{
				id: 'w1',
				name: 'Wallet One',
				displaySubtitle: 'v2',
				href: '/hub/wallets/org/w1'
			}
		]);
		expect(view.columns).toEqual([
			{
				id: 'i1',
				name: 'Issuer One',
				displaySubtitle: 'Issuer sub',
				href: '/hub/credential_issuers/org/i1'
			}
		]);
		expect(view.cells.get('w1:i1')).toMatchObject({ status: 'stable' });
	});

	it('enriches path-based columns from conformance standards', () => {
		const checkPath = 'std/ver/suite/my-check';
		const view = toViewMatrix(
			minimalMatrix({
				column: {
					hub_collection: 'conformance-checks',
					path_based: true
				},
				columns: [
					{
						id: checkPath,
						name: 'Placeholder',
						path: checkPath
					}
				],
				cells: []
			}),
			viewOptions
		);

		expect(view.columns[0]).toEqual({
			id: checkPath,
			name: 'my-check',
			displaySubtitle: 'Suite Name',
			avatar_url: 'https://example.com/logo.png',
			href: `/hub/conformance-checks/${checkPath}`
		});
	});
});
