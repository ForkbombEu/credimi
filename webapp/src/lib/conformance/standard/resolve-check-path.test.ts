// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { Standard } from '../types';

import { resolveCheckPathFromNest } from './resolve-check-path';

const nest: Standard[] = [
	{
		uid: 'fcaf',
		name: 'FCAF',
		versions: [
			{
				uid: 'v1',
				name: 'v1',
				suites: [
					{
						uid: 'rp',
						name: 'RP',
						homepage: '',
						repository: '',
						help: '',
						description: '',
						logo: 'logo.svg',
						members: [{ path: 'fcaf/v1/rp/t1.yaml', title: 'T1', file: 't1.yaml' }]
					}
				]
			}
		]
	}
];

describe('resolveCheckPathFromNest', () => {
	it('returns nest nodes for a four-segment check id', () => {
		const resolved = resolveCheckPathFromNest(nest, 'fcaf/v1/rp/t1');
		expect(resolved).toEqual({
			standard: nest[0],
			version: nest[0].versions[0],
			suite: nest[0].versions[0].suites[0],
			test: 't1'
		});
	});

	it('returns null for wrong segment count', () => {
		expect(resolveCheckPathFromNest(nest, 'fcaf/v1/rp')).toBeNull();
	});

	it('returns null when a nest node is missing', () => {
		expect(resolveCheckPathFromNest(nest, 'fcaf/v1/missing/t1')).toBeNull();
	});
});
