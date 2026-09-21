// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ConformanceCheckRecord } from './record';

import { nestChecks } from './nest';

function check(
	partial: Pick<ConformanceCheckRecord, 'path' | 'standard' | 'version' | 'suite' | 'file'> & {
		title?: string;
		id?: string;
	}
): ConformanceCheckRecord {
	return {
		id: partial.id ?? partial.path,
		title: partial.title ?? partial.file,
		visible_in: ['manual', 'pipeline'],
		protocol: '',
		sut: '',
		role: '',
		provider: '',
		...partial
	};
}

describe('nestChecks', () => {
	it('groups flat records into standards → versions → suites with path identity', () => {
		const nested = nestChecks([
			check({
				path: 'openid4vp_wallet/1.0/openid_conformance_suite/a',
				standard: 'openid4vp_wallet',
				version: '1.0',
				suite: 'openid_conformance_suite',
				file: 'a.yaml'
			}),
			check({
				path: 'openid4vp_wallet/1.0/openid_conformance_suite/b',
				standard: 'openid4vp_wallet',
				version: '1.0',
				suite: 'openid_conformance_suite',
				file: 'b.yaml'
			}),
			check({
				path: 'openid4vci_issuer/1.0/suite_x/c',
				standard: 'openid4vci_issuer',
				version: '1.0',
				suite: 'suite_x',
				file: 'c.json'
			})
		]);

		expect(nested).toHaveLength(2);

		const wallet = nested.find((s) => s.uid === 'openid4vp_wallet');
		expect(wallet?.versions).toHaveLength(1);
		expect(wallet?.versions[0]?.uid).toBe('1.0');
		const suite = wallet?.versions[0]?.suites[0];
		expect(suite?.uid).toBe('openid_conformance_suite');
		expect(suite?.files).toEqual(['a.yaml', 'b.yaml']);
		expect(suite?.paths).toEqual([
			'openid4vp_wallet/1.0/openid_conformance_suite/a',
			'openid4vp_wallet/1.0/openid_conformance_suite/b'
		]);

		const issuer = nested.find((s) => s.uid === 'openid4vci_issuer');
		expect(issuer?.versions[0]?.suites[0]?.paths).toEqual(['openid4vci_issuer/1.0/suite_x/c']);
	});

	it('preserves check_id path strings used by pipeline serialize', () => {
		const path = 'openid4vci_wallet/1.0/openid_conformance_suite/wallet-check';
		const nested = nestChecks([
			check({
				path,
				standard: 'openid4vci_wallet',
				version: '1.0',
				suite: 'openid_conformance_suite',
				file: 'wallet-check.yaml'
			})
		]);
		expect(nested[0]?.versions[0]?.suites[0]?.paths[0]).toBe(path);
	});
});
