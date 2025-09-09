// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { LogStatus, type WorkflowLogsProps } from '$wallet-test/_partials/workflow-logs';
import { z } from 'zod';

//

export function getEUDIWWorkflowLogsProps(
	workflowId?: string,
	namespace?: string
): WorkflowLogsProps {
	if (!workflowId || !namespace) {
		throw new Error('missing workflowId or namespace');
	}

	return {
		subscriptionSuffix: 'eudiw-logs',
		startSignal: 'start-eudiw-check-signal',
		stopSignal: 'stop-eudiw-check-signal',
		workflowId,
		namespace,
		logTransformer: (rawLog) => {
			const data = LogsSchema.parse(rawLog);
			return {
				time: data.timestamp,
				message: data.event + '\n' + data.cause,
				status: LogStatus.INFO,
				rawLog
			};
		}
	};
}

const LogsSchema = z
	.object({
		actor: z.string(),
		event: z.string(),
		cause: z.string().optional(),
		timestamp: z.number().optional()
	})
	.passthrough();
