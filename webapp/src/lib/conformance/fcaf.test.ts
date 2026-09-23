// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { fcafTestIdFromPath, toFcafCatalogEntry } from './client';
import type { ConformanceCheckRecord } from './record';

const sample: ConformanceCheckRecord = {
	id: 'abc123abc123abc',
	path: 'fcaf/wallet_solution/relying_party/WS_RP_DM_Example_001',
	title: 'Example FCAF test',
	fs_standard: 'fcaf',
	fs_version: 'wallet_solution',
	suite: 'relying_party',
	file: 'WS_RP_DM_Example_001.yaml',
	visible_in: ['pipeline'],
	protocol: '',
	sut: 'wallet_solution',
	role: 'relying_party',
	provider: 'fcaf',
	standard: 'openid4vp',
	component: 'wallet',
	version: ''
};

describe('fcaf catalog mapping', () => {
	it('uses the final path segment as the FCAF test id', () => {
		expect(fcafTestIdFromPath(sample.path)).toBe('WS_RP_DM_Example_001');
	});

	it('maps catalog rows to picker entries', () => {
		expect(toFcafCatalogEntry(sample)).toEqual({
			id: 'WS_RP_DM_Example_001',
			title: 'Example FCAF test'
		});
	});
});
