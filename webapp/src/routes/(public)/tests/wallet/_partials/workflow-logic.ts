// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { z } from 'zod';

enum LogStatus {
	'SUCCESS',
	'ERROR',
	'FAILED',
	'FAILURE',
	'WARNING',
	'INFO',
	'INTERRUPTED'
}
export const WorkflowLogOpenIdSchema = z
	.object({
		_id: z.string(),
		msg: z.string(),
		src: z.string(),
		time: z.number().optional(),
		result: z
			.nativeEnum(LogStatus)
			.optional()
	})
	.passthrough();

export const WorkflowLogEudiwSchema = z
	.object({
		actor: z.string(),
		event: z.string(),
		cause: z.string().optional(),
		timestamp: z.number().optional()
	})
	.passthrough();

type WorkflowLog = {
	message: string;
	time: number;
	status: LogStatus;
  rawLog: object;
};

export const GeneralWorkflowLogSchema = WorkflowLogOpenIdSchema.or(WorkflowLogEudiwSchema);

type GeneralWorkflowLog = z.infer<typeof GeneralWorkflowLogSchema>;

export type WorkflowLogsProps = {
	workflowId: string;
	namespace: string;
	subscriptionSuffix: 'openidnet-logs' | 'eudiw-logs';
	startSignal: string;
	stopSignal: string;
	workflowSignalSuffix?: string;
};

type HandlerOptions = WorkflowLogsProps & {
	onUpdate: (data: GeneralWorkflowLog[]) => void;
};

export function createWorkflowLogHandlers({
	workflowId,
	namespace,
	subscriptionSuffix,
	workflowSignalSuffix,
	startSignal,
	stopSignal,
	onUpdate
}: HandlerOptions) {
	const channel = `${workflowId}${subscriptionSuffix}`;
	const signalWorkflowId = workflowSignalSuffix
		? `${workflowId}${workflowSignalSuffix}`
		: workflowId;

	async function startLogs() {
		try {
			await pb.realtime.subscribe(channel, (data) => {
				const parsedResult = z.array(GeneralWorkflowLogSchema).safeParse(data);
				if (!parsedResult.success) {
					throw parsedResult.error;
				}
        const isOpenIdLogs = ;
        const updatedLogs: WorkflowLog[]  = parsedResult.data.map(d => ({
          message: 
        }))
				onUpdate(parsedResult.data);
			});
			await pb.send('/api/compliance/send-temporal-signal', {
				method: 'POST',
				body: {
					workflow_id: signalWorkflowId,
					namespace,
					signal: startSignal
				}
			});
		} catch (e) {
			console.error('Start signal error:', e);
		}
	}

	async function stopLogs() {
		try {
			await pb.realtime.unsubscribe(channel);
			await pb.send('/api/compliance/send-temporal-signal', {
				method: 'POST',
				body: {
					workflow_id: signalWorkflowId,
					namespace,
					signal: stopSignal
				}
			});
		} catch (e) {
			console.error('Stop signal error:', e);
		}
	}

	return { startLogs, stopLogs };
}
