// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { z } from 'zod';

//

export const WorkflowLogEntrySchema = z
	.object({
		_id: z.string(),
		msg: z.string(),
		src: z.string(),
		time: z.number().optional(),
		result: z.enum(['SUCCESS', 'ERROR', 'FAILED', 'WARNING', 'INFO']).optional()
	})
	.passthrough();

export type WorkflowLogEntry = z.infer<typeof WorkflowLogEntrySchema>;

//

export type WorkflowLogsProps = {
	namespace: string;
	workflowId: string;
	workflowType: 'openidnet-logs' | 'eudiw-logs';
	startSignal: string;
	stopSignal: string;
};

type CreateWorkflowLogsHandlersProps = WorkflowLogsProps & {
	onUpdate: (data: WorkflowLogEntry[]) => void;
};

type WorkflowLogsHandlers = {
	startLogs: () => void;
	stopLogs: () => void;
};

export function createWorkflowLogHandlers(
	props: CreateWorkflowLogsHandlersProps
): WorkflowLogsHandlers {
	const { workflowId, namespace, workflowType, startSignal, stopSignal, onUpdate } = props;
	const channel = `${workflowId}${workflowType}`;
	const actualWorkflowId = workflowType === 'openidnet-logs' ? workflowId + '-log' : workflowId;

	function startLogs() {
		pb.realtime
			.subscribe(channel, (data) => {
				const logs = z.array(WorkflowLogEntrySchema).safeParse(data);
				if (logs.success == false) throw new Error(logs.error.message);
				onUpdate(logs.data);
			})
			.catch((e) => console.error('Subscription error:', e));

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: actualWorkflowId,
				namespace,
				signal: startSignal
			}
		}).catch((e) => console.error('Start signal error:', e));
	}

	function stopLogs() {
		pb.realtime.unsubscribe(channel).catch((e) => console.error('Unsubscribe error:', e));

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: actualWorkflowId,
				namespace,
				signal: stopSignal
			}
		}).catch((e) => console.error('Stop signal error:', e));
	}

	return { startLogs, stopLogs };
}
