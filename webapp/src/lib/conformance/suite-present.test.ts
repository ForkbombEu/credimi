// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { entities } from '$lib/global/entities';

import {
	displayNameFromUid,
	displayStandardName,
	entityForComponent,
	suiteHubHref,
	suiteLogo,
	suiteSubtitle,
	suiteTitle
} from './suite-present';

describe('displayNameFromUid', () => {
	it('humanizes underscore and hyphen UIDs', () => {
		expect(displayNameFromUid('openid4vp_wallet')).toBe('Openid4vp Wallet');
		expect(displayNameFromUid('draft-24')).toBe('Draft 24');
		expect(displayNameFromUid('1.0')).toBe('1.0');
	});
});

describe('displayStandardName', () => {
	it('uses authored product-standard labels when present', () => {
		expect(displayStandardName('openid4vp')).toBe('OpenID4VP');
		expect(displayStandardName('openid4vci')).toBe('OpenID4VCI');
	});

	it('falls back to humanized uid for unknown standards', () => {
		expect(displayStandardName('vlei')).toBe('Vlei');
	});
});

describe('suiteTitle', () => {
	it('prefers trimmed suite_name over humanized uid', () => {
		expect(suiteTitle({ suite_name: '  EWC  ', suite: 'ewc' })).toBe('EWC');
	});

	it('falls back to humanized suite uid when name is empty', () => {
		expect(suiteTitle({ suite_name: '   ', suite: 'openid_conformance_suite' })).toBe(
			'Openid Conformance Suite'
		);
	});
});

describe('suiteSubtitle', () => {
	it('returns trimmed subtitle or undefined when empty', () => {
		expect(suiteSubtitle({ suite_subtitle: '  Interop  ' })).toBe('Interop');
		expect(suiteSubtitle({ suite_subtitle: '   ' })).toBeUndefined();
		expect(suiteSubtitle({ suite_subtitle: '' })).toBeUndefined();
	});
});

describe('suiteLogo', () => {
	it('returns trimmed logo URL or undefined when empty', () => {
		expect(suiteLogo({ suite_logo: '  https://example.test/logo.png  ' })).toBe(
			'https://example.test/logo.png'
		);
		expect(suiteLogo({ suite_logo: '   ' })).toBeUndefined();
	});
});

describe('suiteHubHref', () => {
	it('builds the hub detail path from path_prefix', () => {
		expect(
			suiteHubHref({ path_prefix: 'openid4vp_wallet/1.0/openid_conformance_suite' })
		).toBe('/hub/conformance-checks/openid4vp_wallet/1.0/openid_conformance_suite');
	});
});

describe('entityForComponent', () => {
	it('maps known product components to entity metadata', () => {
		expect(entityForComponent('wallet')).toBe(entities.wallets);
		expect(entityForComponent(' issuer ')).toBe(entities.credential_issuers);
		expect(entityForComponent('verifier')).toBe(entities.verifiers);
	});

	it('returns undefined for unknown components', () => {
		expect(entityForComponent('unknown')).toBeUndefined();
	});
});
