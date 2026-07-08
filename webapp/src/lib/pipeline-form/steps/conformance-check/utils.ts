// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getLastPathSegment } from '../_partials/misc';

const OPENID_WALLET_CONFORMANCE_SUITE_PREFIXES = [
	'openid4vci_wallet/1.0/openid_conformance_suite',
	'openid4vp_wallet/1.0/openid_conformance_suite'
] as const;

export function getTestName(test: string): string {
	return getLastPathSegment(test);
}

export function isOpenIdWalletTest(test: string) {
	return test.startsWith('openid4vci_wallet') || test.startsWith('openid4vp_wallet');
}

export function isOpenIdWalletConformanceSuiteTest(test: string) {
	return OPENID_WALLET_CONFORMANCE_SUITE_PREFIXES.some((prefix) => test.startsWith(prefix));
}
