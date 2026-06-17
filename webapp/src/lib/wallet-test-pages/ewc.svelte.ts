// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { LogStatus, type WorkflowLogsProps } from '$lib/workflows/workflow-logs';
import { z } from 'zod/v3';

import type { Getter } from '@/utils/types';

import { pb } from '@/pocketbase';
import { warn } from '@/utils/other';

//

export function setupEWCConnections(
	workflowId: Getter<string | undefined>,
	namespace: Getter<string | undefined>
) {
	$effect(() => {
		const _workflowId = workflowId();
		const _namespace = namespace();
		if (!_workflowId || !_namespace) return;

		function unload() {
			if (!_workflowId || !_namespace) return;
			closeEWCConnections(_workflowId, _namespace);
		}

		window.addEventListener('beforeunload', unload);

		setTimeout(() => {
			pb.send('/api/compliance/send-temporal-signal', {
				method: 'POST',
				body: {
					workflow_id: _workflowId,
					namespace: _namespace,
					signal: 'start-ewc-check-signal'
				}
			}).catch((err) => {
				warn(err);
			});
		}, 2000);

		return () => {
			window.removeEventListener('beforeunload', unload);
			unload();
		};
	});
}

function closeEWCConnections(workflowId: string, namespace: string) {
	pb.send('/api/compliance/send-temporal-signal', {
		method: 'POST',
		body: {
			workflow_id: workflowId,
			namespace: namespace,
			signal: 'stop-ewc-check-signal'
		}
	}).catch((err) => {
		warn(err);
	});
}

export function getEWCWorkflowLogsProps(
	workflowId?: string,
	namespace?: string
): WorkflowLogsProps {
	if (!workflowId || !namespace) {
		throw new Error('missing workflowId or namespace');
	}

	return {
		subscriptionSuffix: 'ewc-logs',
		startSignal: 'start-ewc-check-signal',
		stopSignal: 'stop-ewc-check-signal',
		workflowSignalSuffix: '-status',
		workflowId,
		namespace,
		logTransformer: (rawLog) => {
			const data = LogsSchema.parse(rawLog);
			return {
				time: data.timestamp ? Date.parse(data.timestamp) : undefined,
				message: data.message,
				status: levelToStatus(data.level),
				rawLog
			};
		}
	};
}

function levelToStatus(level: string | undefined): LogStatus {
	switch (level?.toLowerCase()) {
		case 'error':
			return LogStatus.ERROR;
		case 'warn':
		case 'warning':
			return LogStatus.WARNING;
		default:
			return LogStatus.INFO;
	}
}

const LogsSchema = z
	.object({
		level: z.string().optional(),
		message: z.string().optional(),
		step: z.number().optional(),
		timestamp: z.string().optional()
	})
	.passthrough();
