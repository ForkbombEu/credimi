// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { String } from 'effect';

export type WalletTestParams = {
	qr: string;
	workflowId: string;
};

export function getWalletTestParams(url: URL): WalletTestParams | Error {
	const qr = url.searchParams.get('qr');
	const workflowId = url.searchParams.get('workflow-id');

	const hasError = !qr || String.isEmpty(qr) || !workflowId || String.isEmpty(workflowId);

	if (hasError) {
		return new Error('Invalid params');
	}

	return {
		qr,
		workflowId
	};
}
