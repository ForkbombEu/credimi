// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from '$lib/conformance/types';

import { describe, expect, it } from 'vitest';

import { m } from '@/i18n';

import type { InteropMatrixResponse } from './types';

import {
	buildVisibleMatrix,
	cellKey,
	hubHref,
	subtitleOrVersion,
	toViewMatrix
} from './to-view-matrix';

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
		row: { hub_collection: 'wallets', path_based: false, tiered: false },
		column: { hub_collection: 'credential_issuers', path_based: false, tiered: false },
		row_groups: [],
		row_leaves: [
			{
				id: 'w1',
				name: 'Wallet One',
				path: 'org/w1',
				version_label: 'v2'
			}
		],
		column_groups: [],
		column_leaves: [
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
				row_tier: 'leaf',
				column_tier: 'leaf',
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

const viewOptions = {
	standards,
	expandedRowGroups: new Set<string>(),
	expandedColumnGroups: new Set<string>()
};

function tieredWalletIssuerMatrix(): InteropMatrixResponse {
	return {
		row: { hub_collection: 'wallets', path_based: false, tiered: true },
		column: { hub_collection: 'credential_issuers', path_based: false, tiered: true },
		row_groups: [
			{
				id: 'wallet-g',
				name: 'Wallet Group',
				path: 'org/wallet',
				child_count: 2,
				avatar_url: 'https://example.com/w.png'
			}
		],
		row_leaves: [
			{
				id: 'version-1',
				parent_id: 'wallet-g',
				name: 'Wallet Group',
				path: 'org/wallet/v1',
				version_label: 'v1.0'
			},
			{
				id: 'wallet-g::__no_version__',
				parent_id: 'wallet-g',
				name: 'Wallet Group',
				path: 'org/wallet',
				version_label: null
			}
		],
		column_groups: [
			{
				id: 'issuer-g',
				name: 'Issuer Group',
				path: 'org/issuer',
				child_count: 1
			}
		],
		column_leaves: [
			{
				id: 'cred-1',
				parent_id: 'issuer-g',
				name: 'Credential',
				path: 'org/issuer/c1',
				subtitle: 'Cred sub'
			}
		],
		cells: [
			{
				row_id: 'wallet-g',
				column_id: 'issuer-g',
				row_tier: 'group',
				column_tier: 'group',
				pipeline_count: 2,
				total_runs: 20,
				total_successes: 18,
				success_rate: 90,
				status: 'stable'
			},
			{
				row_id: 'version-1',
				column_id: 'cred-1',
				row_tier: 'leaf',
				column_tier: 'leaf',
				pipeline_count: 1,
				total_runs: 5,
				total_successes: 4,
				success_rate: 80,
				status: 'flaky'
			}
		]
	};
}

describe('hubHref', () => {
	it('joins hub collection and entity path', () => {
		expect(
			hubHref({ hub_collection: 'wallets', path_based: false, tiered: false }, 'org/w1')
		).toBe('/hub/wallets/org/w1');
	});

	it('uses conformance-checks segment for path-based columns', () => {
		expect(
			hubHref(
				{ hub_collection: 'conformance-checks', path_based: true, tiered: true },
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

describe('cellKey', () => {
	it('joins tier and id for row and column', () => {
		expect(cellKey('group', 'r1', 'leaf', 'c2')).toBe('group:r1:leaf:c2');
	});
});

describe('buildVisibleMatrix', () => {
	it('builds corner label, entity hrefs, subtitles, and tier-aware cell lookup', () => {
		const view = buildVisibleMatrix(minimalMatrix(), viewOptions);

		expect(view.cornerLabel).toBe(
			m.interop_matrix_corner_label({ row: m.Wallet(), column: m.Issuer() })
		);
		expect(view.rows).toEqual([
			{
				id: 'w1',
				name: 'Wallet One',
				displaySubtitle: 'v2',
				href: '/hub/wallets/org/w1',
				tier: 'leaf',
				nested: false
			}
		]);
		expect(view.columns).toEqual([
			{
				id: 'i1',
				name: 'Issuer One',
				displaySubtitle: 'Issuer sub',
				href: '/hub/credential_issuers/org/i1',
				tier: 'leaf',
				nested: false
			}
		]);
		expect(view.cells.get(cellKey('leaf', 'w1', 'leaf', 'i1'))).toMatchObject({
			status: 'stable'
		});
	});

	it('shows group rows when tiered axis is collapsed', () => {
		const view = buildVisibleMatrix(tieredWalletIssuerMatrix(), viewOptions);

		expect(view.rows).toEqual([
			{
				id: 'wallet-g',
				name: 'Wallet Group',
				href: '/hub/wallets/org/wallet',
				avatar_url: 'https://example.com/w.png',
				tier: 'group',
				child_count: 2
			}
		]);
		expect(view.columns).toEqual([
			{
				id: 'issuer-g',
				name: 'Issuer Group',
				href: '/hub/credential_issuers/org/issuer',
				tier: 'group',
				child_count: 1
			}
		]);
		expect(view.cells.get(cellKey('group', 'wallet-g', 'group', 'issuer-g'))).toMatchObject({
			status: 'stable'
		});
	});

	it('shows group row plus nested leaves when row group is expanded', () => {
		const view = buildVisibleMatrix(tieredWalletIssuerMatrix(), {
			...viewOptions,
			expandedRowGroups: new Set(['wallet-g'])
		});

		expect(view.rows.map((row) => row.id)).toEqual([
			'wallet-g',
			'version-1',
			'wallet-g::__no_version__'
		]);
		expect(view.rows[0]?.tier).toBe('group');
		expect(view.rows[1]?.nested).toBe(true);
		expect(view.rows[1]?.displaySubtitle).toBe('v1.0');
		expect(view.rows[2]?.displaySubtitle).toBe(m.interop_matrix_version_undefined());
	});

	it('shows group column plus nested leaves when column group is expanded', () => {
		const view = buildVisibleMatrix(tieredWalletIssuerMatrix(), {
			...viewOptions,
			expandedColumnGroups: new Set(['issuer-g'])
		});

		expect(view.columns.map((column) => column.id)).toEqual(['issuer-g', 'cred-1']);
		expect(view.columns[0]?.tier).toBe('group');
		expect(view.columns[1]?.tier).toBe('leaf');
		expect(view.columns[1]?.nested).toBe(true);
	});

	it('resolves cells with tier-aware keys when both axes expanded', () => {
		const view = buildVisibleMatrix(tieredWalletIssuerMatrix(), {
			...viewOptions,
			expandedRowGroups: new Set(['wallet-g']),
			expandedColumnGroups: new Set(['issuer-g'])
		});

		expect(view.cells.get(cellKey('leaf', 'version-1', 'leaf', 'cred-1'))).toMatchObject({
			status: 'flaky'
		});
	});

	it('enriches path-based columns from conformance standards', () => {
		const checkPath = 'std/ver/suite/my-check';
		const view = buildVisibleMatrix(
			minimalMatrix({
				column: {
					hub_collection: 'conformance-checks',
					path_based: true,
					tiered: false
				},
				column_groups: [],
				column_leaves: [
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
			href: `/hub/conformance-checks/${checkPath}`,
			tier: 'leaf',
			nested: false
		});
	});
});

describe('toViewMatrix', () => {
	it('delegates to buildVisibleMatrix', () => {
		const fromAlias = toViewMatrix(minimalMatrix(), viewOptions);
		const fromBuild = buildVisibleMatrix(minimalMatrix(), viewOptions);
		expect(fromAlias).toEqual(fromBuild);
	});
});
