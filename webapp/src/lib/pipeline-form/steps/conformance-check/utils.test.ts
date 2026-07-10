// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { isOpenIdWalletConformanceSuiteTest, isOpenIdWalletTest } from './utils';

describe('isOpenIdWalletTest', () => {
	it('matches any openid wallet standard prefix', () => {
		expect(isOpenIdWalletTest('openid4vci_wallet/foo')).toBe(true);
		expect(isOpenIdWalletTest('openid4vp_wallet/draft-24/bar')).toBe(true);
	});

	it('does not match issuer or verifier checks', () => {
		expect(isOpenIdWalletTest('openid4vci_issuer/foo')).toBe(false);
	});
});

describe('isOpenIdWalletConformanceSuiteTest', () => {
	it('matches only the 1.0 openid_conformance_suite wallet paths', () => {
		expect(
			isOpenIdWalletConformanceSuiteTest(
				'openid4vci_wallet/1.0/openid_conformance_suite/wallet-check'
			)
		).toBe(true);
		expect(
			isOpenIdWalletConformanceSuiteTest(
				'openid4vp_wallet/1.0/openid_conformance_suite/wallet-check'
			)
		).toBe(true);
	});

	it('does not match other wallet suites or standards', () => {
		expect(isOpenIdWalletConformanceSuiteTest('openid4vci_wallet/foo')).toBe(false);
		expect(
			isOpenIdWalletConformanceSuiteTest('openid4vci_wallet/draft-15/ewc/wallet-check')
		).toBe(false);
	});
});
