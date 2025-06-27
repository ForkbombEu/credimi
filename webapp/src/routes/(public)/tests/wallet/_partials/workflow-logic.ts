// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { z } from 'zod';

//

export enum LogStatus {
	SUCCESS = 'Success',
	ERROR = 'Error',
	FAILED = 'Failed',
	FAILURE = 'Failure',
	WARNING = 'Warning',
	INFO = 'Info',
	INTERRUPTED = 'Interrupted'
}

export type WorkflowLog = {
	message?: string;
	time?: number;
	status?: LogStatus;
	rawLog: unknown;
};

export type WorkflowLogsProps = {
	workflowId: string;
	namespace: string;
	subscriptionSuffix: 'openidnet-logs' | 'eudiw-logs';
	startSignal: string;
	stopSignal: string;
	workflowSignalSuffix?: string;
	logTransformer: (data: unknown) => WorkflowLog;
};

type HandlerOptions = WorkflowLogsProps & {
	onUpdate: (data: WorkflowLog[]) => void;
};

export function createWorkflowLogHandlers(props: HandlerOptions) {
	const {
		workflowId,
		namespace,
		subscriptionSuffix,
		workflowSignalSuffix,
		startSignal,
		stopSignal,
		onUpdate,
		logTransformer
	} = props;

	const channel = `${workflowId}${subscriptionSuffix}`;
	const signalWorkflowId = workflowSignalSuffix
		? `${workflowId}${workflowSignalSuffix}`
		: workflowId;

	async function startLogs() {
		try {
			await pb.realtime.subscribe(channel, (data) => {
				const parseResult = z.array(z.unknown()).safeParse(data);
				if (!parseResult.success) throw new Error('Unexpected data shape');
				onUpdate(
					parseResult.data.reverse().map((datum) => {
						try {
							return logTransformer(datum);
						} catch (e) {
							console.error('Log transformer error:', e);
							return { status: LogStatus.INFO, rawLog: datum };
						}
					})
				);
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
