// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getWalletTestParams } from './_partials';

const ALLOWED_TESTS = new Set(['openidnet', 'ewc', 'eudiw']);
const LOGS_ENABLED_TESTS = new Set(['openidnet', 'ewc']);
const STATUS_BUTTONS_ENABLED_TESTS = new Set(['openidnet', 'eudiw']);

const names: Record<string, string> = {
	openidnet: "OpenID",
	ewc: "EWC",
	eudiw: "EU Digital Identity Wallet",
}

export const load = ({ url, params }) => {
	if (!ALLOWED_TESTS.has(params.mode)) {
		error(404);
	}
	const walletTestParams = getWalletTestParams(url);
	if (!walletTestParams.workflowId) {
		error(404);
	}

	const showLogs = LOGS_ENABLED_TESTS.has(params.mode);
	const showStatusButtons = STATUS_BUTTONS_ENABLED_TESTS.has(params.mode);
	const testName = names[params.mode]
	const isCloseConnection = params.mode === "ewc";

	return { ...walletTestParams, showLogs, showStatusButtons, testName, isCloseConnection };
};
