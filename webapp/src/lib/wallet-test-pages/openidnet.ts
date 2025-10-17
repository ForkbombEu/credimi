// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { LogStatus, type WorkflowLogsProps } from '$lib/workflows/workflow-logs';
import { z } from 'zod';

//

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

const LogsSchema = z
	.object({
		_id: z.string(),
		msg: z.string(),
		src: z.string(),
		time: z.number().optional(),
		result: z.nativeEnum(LogStatus).optional()
	})
	.passthrough();
