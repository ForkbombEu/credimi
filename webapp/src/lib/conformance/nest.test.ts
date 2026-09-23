// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ConformanceCheckRecord } from './record';

import { displayNameFromUid, nestChecks, titleForCheckPath } from './nest';

function check(
	partial: Pick<ConformanceCheckRecord, 'path' | 'standard' | 'version' | 'suite' | 'file'> &
		Partial<ConformanceCheckRecord>
): ConformanceCheckRecord {
	return {
		id: partial.id ?? partial.path,
		title: partial.title ?? partial.file,
		visible_in: ['manual', 'pipeline'],
		protocol: '',
		sut: '',
		role: '',
		provider: '',
		norm_standard: '',
		component: '',
		norm_version: '',
		suite_name: '',
		suite_homepage: '',
		suite_repository: '',
		suite_help: '',
		suite_description: '',
		suite_logo: '',
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
				file: 'a.yaml',
				title: 'Check A'
			}),
			check({
				path: 'openid4vp_wallet/1.0/openid_conformance_suite/b',
				standard: 'openid4vp_wallet',
				version: '1.0',
				suite: 'openid_conformance_suite',
				file: 'b.yaml',
				title: 'Check B'
			}),
			check({
				path: 'openid4vci_issuer/1.0/suite_x/c',
				standard: 'openid4vci_issuer',
				version: '1.0',
				suite: 'suite_x',
				file: 'c.json',
				title: 'Issuer Check C'
			})
		]);

		expect(nested).toHaveLength(2);

		const wallet = nested.find((s) => s.uid === 'openid4vp_wallet');
		expect(wallet?.name).toBe('Openid4vp Wallet');
		expect(wallet?.versions).toHaveLength(1);
		expect(wallet?.versions[0]?.uid).toBe('1.0');
		expect(wallet?.versions[0]?.name).toBe('1.0');
		const suite = wallet?.versions[0]?.suites[0];
		expect(suite?.uid).toBe('openid_conformance_suite');
		expect(suite?.name).toBe('Openid Conformance Suite');
		expect(suite?.files).toEqual(['a.yaml', 'b.yaml']);
		expect(suite?.paths).toEqual([
			'openid4vp_wallet/1.0/openid_conformance_suite/a',
			'openid4vp_wallet/1.0/openid_conformance_suite/b'
		]);
		expect(suite?.titles).toEqual(['Check A', 'Check B']);

		const issuer = nested.find((s) => s.uid === 'openid4vci_issuer');
		expect(issuer?.name).toBe('Openid4vci Issuer');
		expect(issuer?.versions[0]?.suites[0]?.paths).toEqual(['openid4vci_issuer/1.0/suite_x/c']);
		expect(issuer?.versions[0]?.suites[0]?.titles).toEqual(['Issuer Check C']);
	});

	it('prefers denormalized suite metadata over humanized UIDs', () => {
		const nested = nestChecks([
			check({
				path: 'openid4vp_wallet/draft-23/ewc/check_one',
				standard: 'openid4vp_wallet',
				version: 'draft-23',
				suite: 'ewc',
				file: 'check_one.yaml',
				title: 'Check One',
				suite_name: 'EWC Interoperability Test Bed',
				suite_homepage: 'https://eudiwalletconsortium.org/',
				suite_repository: 'https://github.com/EWC-consortium',
				suite_help: 'https://example.test/help',
				suite_description: 'EWC ITB',
				suite_logo: 'https://example.test/ewc.png'
			})
		]);
		const suite = nested[0]?.versions[0]?.suites[0];
		expect(suite?.name).toBe('EWC Interoperability Test Bed');
		expect(suite?.homepage).toBe('https://eudiwalletconsortium.org/');
		expect(suite?.repository).toBe('https://github.com/EWC-consortium');
		expect(suite?.help).toBe('https://example.test/help');
		expect(suite?.description).toBe('EWC ITB');
		expect(suite?.logo).toBe('https://example.test/ewc.png');
	});

	it('preserves check_id path strings used by pipeline serialize', () => {
		const path = 'openid4vci_wallet/1.0/openid_conformance_suite/wallet-check';
		const nested = nestChecks([
			check({
				path,
				standard: 'openid4vci_wallet',
				version: '1.0',
				suite: 'openid_conformance_suite',
				file: 'wallet-check.yaml',
				title: 'Wallet Check'
			})
		]);
		expect(nested[0]?.versions[0]?.suites[0]?.paths[0]).toBe(path);
		expect(nested[0]?.versions[0]?.suites[0]?.titles[0]).toBe('Wallet Check');
	});

	it('leaves logo and description empty when catalog rows lack them', () => {
		const nested = nestChecks([
			check({
				path: 'vlei/version/suite/one',
				standard: 'vlei',
				version: 'version',
				suite: 'suite',
				file: 'one.yaml',
				title: 'One'
			})
		]);
		const standard = nested[0];
		const suite = standard?.versions[0]?.suites[0];
		expect(standard?.description).toBe('');
		expect(standard?.standard_url).toBe('');
		expect(suite?.description).toBe('');
		expect(suite?.homepage).toBe('');
		expect(suite?.logo).toBeUndefined();
	});
});

describe('displayNameFromUid', () => {
	it('humanizes underscore and hyphen UIDs', () => {
		expect(displayNameFromUid('openid4vp_wallet')).toBe('Openid4vp Wallet');
		expect(displayNameFromUid('draft-24')).toBe('Draft 24');
		expect(displayNameFromUid('1.0')).toBe('1.0');
	});
});

describe('titleForCheckPath', () => {
	it('resolves by full path or stem against suite titles', () => {
		const suite = {
			paths: [
				'openid4vp_wallet/1.0/openid_conformance_suite/happy_flow',
				'openid4vp_wallet/1.0/openid_conformance_suite/alt'
			],
			titles: ['Happy Flow', 'Alternate']
		};
		expect(titleForCheckPath(suite, suite.paths[0]!)).toBe('Happy Flow');
		expect(titleForCheckPath(suite, 'happy_flow')).toBe('Happy Flow');
		expect(titleForCheckPath(suite, 'missing')).toBe('missing');
	});
});
