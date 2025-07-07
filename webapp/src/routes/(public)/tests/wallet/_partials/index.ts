// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { String } from 'effect';

export type WalletTestParams = {
	qr: string | undefined;
	workflowId: string | undefined;
	namespace: string | undefined;
};

export function getWalletTestParams(url: URL): WalletTestParams {
	const qr = url.searchParams.get('qr');
	const workflowId = url.searchParams.get('workflow-id');
	const namespace = url.searchParams.get('namespace');

	const hasQr = qr && !String.isEmpty(qr);
	const hasWorkflowId = workflowId && !String.isEmpty(workflowId);
	const hasNamespace = namespace && !String.isEmpty(namespace);

	return {
		qr: hasQr ? qr : undefined,
		workflowId: hasWorkflowId ? workflowId : undefined,
		namespace: hasNamespace ? namespace : undefined,
	};
}
