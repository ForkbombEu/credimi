// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { LogStatus, type WorkflowLogsProps } from '$lib/workflows/workflow-logs';
import { z } from 'zod/v3';

//

export type OpenIDConformanceStandard =
	| 'openid4vci_issuer'
	| 'openid4vp_verifier'
	| 'openid4vci_wallet'
	| 'openid4vp_wallet';

const openIDConformanceStandards = new Set<OpenIDConformanceStandard>([
	'openid4vci_issuer',
	'openid4vp_verifier',
	'openid4vci_wallet',
	'openid4vp_wallet'
]);

export function isOpenIDConformanceStandard(
	standard: string | undefined
): standard is OpenIDConformanceStandard {
	return (
		standard !== undefined &&
		openIDConformanceStandards.has(standard as OpenIDConformanceStandard)
	);
}

export function getOpenIDConformanceWorkflowLogsProps(
	workflowId: string | undefined,
	namespace: string | undefined,
	standard: OpenIDConformanceStandard
): WorkflowLogsProps {
	switch (standard) {
		case 'openid4vci_issuer':
			return getOpenID4VCIIssuerWorkflowLogsProps(workflowId, namespace);
		case 'openid4vp_verifier':
			return getOpenID4VPVerifierWorkflowLogsProps(workflowId, namespace);
		case 'openid4vci_wallet':
		case 'openid4vp_wallet':
			return getOpenIDNetWorkflowLogsProps(workflowId, namespace);
	}
}

export function getOpenIDNetWorkflowLogsProps(
	workflowId?: string,
	namespace?: string
): WorkflowLogsProps {
	if (!workflowId || !namespace) {
		throw new Error('missing workflowId or namespace');
	}

	return {
		subscriptionSuffix: 'openidnet-logs',
		startSignal: 'start-openidnet-check-log-update',
		stopSignal: 'stop-openidnet-check-log-update',
		workflowSignalSuffix: '-log',
		workflowId,
		namespace,
		logTransformer: (rawLog) => {
			const data = LogsSchema.parse(rawLog);
			return {
				time: data.time,
				message: data.msg,
				status: data.result,
				rawLog
			};
		}
	};
}

export function getOpenID4VCIIssuerWorkflowLogsProps(
	workflowId?: string,
	namespace?: string
): WorkflowLogsProps {
	if (!workflowId || !namespace) {
		throw new Error('missing workflowId or namespace');
	}

	return {
		subscriptionSuffix: 'openidnet-logs',
		startSignal: 'start-openid4vci-issuer-log-update',
		stopSignal: 'stop-openid4vci-issuer-log-update',
		workflowId,
		namespace,
		logTransformer: (rawLog) => {
			const data = LogsSchema.parse(rawLog);
			return {
				time: data.time,
				message: data.msg,
				status: data.result,
				rawLog
			};
		}
	};
}

export function getOpenID4VPVerifierWorkflowLogsProps(
	workflowId?: string,
	namespace?: string
): WorkflowLogsProps {
	if (!workflowId || !namespace) {
		throw new Error('missing workflowId or namespace');
	}

	return {
		subscriptionSuffix: 'openidnet-logs',
		startSignal: 'start-openid4vp-verifier-log-update',
		stopSignal: 'stop-openid4vp-verifier-log-update',
		workflowId,
		namespace,
		logTransformer: (rawLog) => {
			const data = LogsSchema.parse(rawLog);
			return {
				time: data.time,
				message: data.msg,
				status: data.result,
				rawLog
			};
		}
	};
}

const LogsSchema = z
	.object({
		_id: z.string(),
		msg: z.string(),
		src: z.string(),
		time: z.number().optional(),
		result: z.nativeEnum(LogStatus).optional()
	})
	.passthrough();
