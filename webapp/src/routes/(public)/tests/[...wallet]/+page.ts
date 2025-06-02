// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getWalletTestParams } from './_partials';

const ALLOWED_TESTS = new Set(['wallet', 'ewc', 'eudiw']);
const LOGS_ENABLED_TESTS = new Set(['wallet', 'ewc']);
const STATUS_BUTTONS_ENABLED_TESTS = new Set(['wallet', 'eudiw']);

export const load = ({ url }) => {
	const walletTestParams = getWalletTestParams(url);
	const requestedWalletTest = isAllowedTestUrl(url);
	if (!walletTestParams.workflowId || !requestedWalletTest) {
		error(404);
	}
	
	const showLogs = LOGS_ENABLED_TESTS.has(requestedWalletTest);
	const showStatusButtons = STATUS_BUTTONS_ENABLED_TESTS.has(requestedWalletTest);
	const testName =
		requestedWalletTest === 'wallet'
			? requestedWalletTest.charAt(0).toUpperCase() + requestedWalletTest.slice(1)
			: requestedWalletTest.toUpperCase();

	return { ...walletTestParams, showLogs, showStatusButtons, testName };
};

function isAllowedTestUrl(url: URL): string {
	const segments = url.pathname.match(/[^/]+/g) || [];
	const last = segments[segments.length - 1];
	return ALLOWED_TESTS.has(last) ? last : '';
}
