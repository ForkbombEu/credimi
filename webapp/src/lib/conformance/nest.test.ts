// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ConformanceSuiteRecord } from './record';

import { displayNameFromUid, nestSuites, titleForCheckPath } from './nest';

function suite(
	partial: Pick<
		ConformanceSuiteRecord,
		'fs_standard' | 'fs_version' | 'suite' | 'path_prefix' | 'members'
	> &
		Partial<ConformanceSuiteRecord>
): ConformanceSuiteRecord {
	return {
		id: partial.id ?? partial.path_prefix,
		standard: partial.standard ?? partial.fs_standard,
		component: partial.component ?? '',
		component_rank: partial.component_rank ?? 9,
		version: partial.version ?? partial.fs_version,
		suite: partial.suite,
		provider: partial.provider ?? '',
		suite_name: partial.suite_name ?? '',
		suite_homepage: partial.suite_homepage ?? '',
		suite_repository: partial.suite_repository ?? '',
		suite_help: partial.suite_help ?? '',
		suite_description: partial.suite_description ?? '',
		suite_logo: partial.suite_logo ?? '',
		check_count: partial.check_count ?? partial.members.length,
		members: partial.members,
		visible_in: partial.visible_in ?? ['manual', 'pipeline'],
		fs_standard: partial.fs_standard,
		fs_version: partial.fs_version,
		path_prefix: partial.path_prefix
	};
}

describe('nestSuites', () => {
	it('groups suite rows into standards → versions → suites on fs_* axes', () => {
		const nested = nestSuites([
			suite({
				fs_standard: 'openid4vp_wallet',
				fs_version: '1.0',
				suite: 'openid_conformance_suite',
				path_prefix: 'openid4vp_wallet/1.0/openid_conformance_suite',
				members: [
					{
						path: 'openid4vp_wallet/1.0/openid_conformance_suite/a',
						title: 'Check A',
						file: 'a.yaml'
					},
					{
						path: 'openid4vp_wallet/1.0/openid_conformance_suite/b',
						title: 'Check B',
						file: 'b.yaml'
					}
				]
			}),
			suite({
				fs_standard: 'openid4vci_issuer',
				fs_version: '1.0',
				suite: 'suite_x',
				path_prefix: 'openid4vci_issuer/1.0/suite_x',
				members: [
					{
						path: 'openid4vci_issuer/1.0/suite_x/c',
						title: 'Issuer Check C',
						file: 'c.json'
					}
				]
			})
		]);

		expect(nested).toHaveLength(2);

		const wallet = nested.find((s) => s.uid === 'openid4vp_wallet');
		expect(wallet?.name).toBe('Openid4vp Wallet');
		expect(wallet?.versions).toHaveLength(1);
		expect(wallet?.versions[0]?.uid).toBe('1.0');
		const nestedSuite = wallet?.versions[0]?.suites[0];
		expect(nestedSuite?.uid).toBe('openid_conformance_suite');
		expect(nestedSuite?.name).toBe('Openid Conformance Suite');
		expect(nestedSuite?.members).toEqual([
			{
				path: 'openid4vp_wallet/1.0/openid_conformance_suite/a',
				title: 'Check A',
				file: 'a.yaml'
			},
			{
				path: 'openid4vp_wallet/1.0/openid_conformance_suite/b',
				title: 'Check B',
				file: 'b.yaml'
			}
		]);

		const issuer = nested.find((s) => s.uid === 'openid4vci_issuer');
		expect(issuer?.versions[0]?.suites[0]?.members).toEqual([
			{
				path: 'openid4vci_issuer/1.0/suite_x/c',
				title: 'Issuer Check C',
				file: 'c.json'
			}
		]);
	});

	it('prefers suite-row display metadata over humanized UIDs', () => {
		const nested = nestSuites([
			suite({
				fs_standard: 'openid4vp_wallet',
				fs_version: 'draft-23',
				suite: 'ewc',
				path_prefix: 'openid4vp_wallet/draft-23/ewc',
				members: [
					{
						path: 'openid4vp_wallet/draft-23/ewc/check_one',
						title: 'Check One',
						file: 'check_one.yaml'
					}
				],
				suite_name: 'EWC Interoperability Test Bed',
				suite_homepage: 'https://eudiwalletconsortium.org/',
				suite_repository: 'https://github.com/EWC-consortium',
				suite_help: 'https://example.test/help',
				suite_description: 'EWC ITB',
				suite_logo: 'https://example.test/ewc.png'
			})
		]);
		const nestedSuite = nested[0]?.versions[0]?.suites[0];
		expect(nestedSuite?.name).toBe('EWC Interoperability Test Bed');
		expect(nestedSuite?.homepage).toBe('https://eudiwalletconsortium.org/');
		expect(nestedSuite?.repository).toBe('https://github.com/EWC-consortium');
		expect(nestedSuite?.help).toBe('https://example.test/help');
		expect(nestedSuite?.description).toBe('EWC ITB');
		expect(nestedSuite?.logo).toBe('https://example.test/ewc.png');
	});

	it('preserves check path strings used by pipeline serialize', () => {
		const path = 'openid4vci_wallet/1.0/openid_conformance_suite/wallet-check';
		const nested = nestSuites([
			suite({
				fs_standard: 'openid4vci_wallet',
				fs_version: '1.0',
				suite: 'openid_conformance_suite',
				path_prefix: 'openid4vci_wallet/1.0/openid_conformance_suite',
				members: [{ path, title: 'Wallet Check', file: 'wallet-check.yaml' }]
			})
		]);
		expect(nested[0]?.versions[0]?.suites[0]?.members[0]?.path).toBe(path);
		expect(nested[0]?.versions[0]?.suites[0]?.members[0]?.title).toBe('Wallet Check');
	});

	it('leaves suite logo and description empty when suite rows lack them', () => {
		const nested = nestSuites([
			suite({
				fs_standard: 'vlei',
				fs_version: 'version',
				suite: 'suite',
				path_prefix: 'vlei/version/suite',
				members: [{ path: 'vlei/version/suite/one', title: 'One', file: 'one.yaml' }]
			})
		]);
		const standard = nested[0];
		const nestedSuite = standard?.versions[0]?.suites[0];
		expect(standard).toMatchObject({ uid: 'vlei', name: 'Vlei' });
		expect(standard).not.toHaveProperty('description');
		expect(standard).not.toHaveProperty('standard_url');
		expect(nestedSuite?.description).toBe('');
		expect(nestedSuite?.homepage).toBe('');
		expect(nestedSuite?.logo).toBeUndefined();
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
	it('resolves by full path or stem against suite members', () => {
		const nestedSuite = {
			members: [
				{
					path: 'openid4vp_wallet/1.0/openid_conformance_suite/happy_flow',
					title: 'Happy Flow',
					file: 'happy_flow.yaml'
				},
				{
					path: 'openid4vp_wallet/1.0/openid_conformance_suite/alt',
					title: 'Alternate',
					file: 'alt.yaml'
				}
			]
		};
		expect(titleForCheckPath(nestedSuite, nestedSuite.members[0]!.path)).toBe('Happy Flow');
		expect(titleForCheckPath(nestedSuite, 'happy_flow')).toBe('Happy Flow');
		expect(titleForCheckPath(nestedSuite, 'missing')).toBe('missing');
	});
});
