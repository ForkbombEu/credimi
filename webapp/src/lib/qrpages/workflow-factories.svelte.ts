// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { LogStatus } from './workflow-types';
import { WorkflowManager, type WorkflowManagerConfig } from './workflow-manager.svelte';

const OpenIdNetLogsSchema = z
	.object({
		_id: z.string(),
		msg: z.string(),
		src: z.string(),
		time: z.number().optional(),
		result: z.nativeEnum(LogStatus).optional()
	})
	.passthrough();

const EudiwLogsSchema = z
	.object({
		actor: z.string(),
		event: z.string(),
		cause: z.string().optional(),
		timestamp: z.number().optional()
	})
	.passthrough();

export function createOpenIdNetWorkflowManager(workflowId: string, namespace: string): WorkflowManager {
	const config: WorkflowManagerConfig = {
		workflowId,
		namespace,
		subscriptionSuffix: 'openidnet-logs',
		startSignal: 'start-openidnet-check-log-update',
		stopSignal: 'stop-openidnet-check-log-update',
		workflowSignalSuffix: '-log',
		logTransformer: (rawLog) => {
			const data = OpenIdNetLogsSchema.parse(rawLog);
			return {
				time: data.time,
				message: data.msg,
				status: data.result,
				rawLog
			};
		}
	};
	return new WorkflowManager(config);
}

export function createEudiwWorkflowManager(workflowId: string, namespace: string): WorkflowManager {
	const config: WorkflowManagerConfig = {
		workflowId,
		namespace,
		subscriptionSuffix: 'eudiw-logs',
		startSignal: 'start-eudiw-check-signal',
		stopSignal: 'stop-eudiw-check-signal',
		logTransformer: (rawLog) => {
			const data = EudiwLogsSchema.parse(rawLog);
			return {
				time: data.timestamp,
				message: data.event + '\n' + data.cause,
				status: LogStatus.INFO,
				rawLog
			};
		}
	};
	return new WorkflowManager(config);
}

export function createEwcWorkflowManager(workflowId: string, namespace: string): WorkflowManager {
	const config: WorkflowManagerConfig = {
		workflowId,
		namespace,
		subscriptionSuffix: 'ewc-logs',
		startSignal: 'start-ewc-check-signal',
		stopSignal: 'stop-ewc-check-signal'
	};
	return new WorkflowManager(config);
}
