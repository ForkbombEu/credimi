// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

import { pb } from '@/pocketbase/index.js';

//

export enum LogStatus {
	SUCCESS = 'SUCCESS',
	ERROR = 'ERROR',
	FAILED = 'FAILED',
	FAILURE = 'FAILURE',
	WARNING = 'WARNING',
	INFO = 'INFO',
	INTERRUPTED = 'INTERRUPTED'
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

	function startLogs() {
		setTimeout(async () => {
			try {
				await pb.realtime.subscribe(channel, (data) => {
					const parseResult = z.array(z.unknown()).safeParse(data);
					if (!parseResult.success) throw new Error('Unexpected data shape');
					onUpdate(
						parseResult.data
							.map((datum) => {
								try {
									return logTransformer(datum);
								} catch (e) {
									console.error('Log transformer error:', e);
									return { status: LogStatus.INFO, rawLog: datum };
								}
							})
							.sort((a, b) => (a.time ?? 0) - (b.time ?? 0))
					);
				});
				await pb.send('/api/compliance/send-temporal-signal', {
					method: 'POST',
					body: {
						workflow_id: signalWorkflowId,
						namespace,
						signal: startSignal
					},
					requestKey: null
				});
			} catch (e) {
				console.error('Start signal error:', e);
			}
		}, 2000);
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
				},
				requestKey: null
			});
		} catch (e) {
			console.error('Stop signal error:', e);
		}
	}

	return { startLogs, stopLogs };
}
