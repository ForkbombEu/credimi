// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import type { WorkflowLog, WorkflowLogsProps } from './workflow-types';

export interface WorkflowManagerConfig {
	workflowId: string;
	namespace: string;
	subscriptionSuffix: string;
	startSignal?: string;
	stopSignal?: string;
	workflowSignalSuffix?: string;
	logTransformer?: (rawLog: unknown) => WorkflowLog;
}

export class WorkflowManager {
	protected workflowId: string;
	protected namespace: string;
	protected config: WorkflowManagerConfig;
	private beforeUnloadHandler: (() => void) | null = null;

	constructor(config: WorkflowManagerConfig) {
		this.config = config;
		this.workflowId = config.workflowId;
		this.namespace = config.namespace;
		
		// Start immediately to maintain backwards compatibility and prevent race conditions
		this.onMount();
		this.setupCleanup();
	}

	private setupCleanup(): void {
		if (typeof window !== 'undefined') {
			this.beforeUnloadHandler = () => this.onDestroy();
			window.addEventListener('beforeunload', this.beforeUnloadHandler);
		}
	}

	destroy(): void {
		this.onDestroy();

		// Remove event listener properly
		if (typeof window !== 'undefined' && this.beforeUnloadHandler) {
			window.removeEventListener('beforeunload', this.beforeUnloadHandler);
			this.beforeUnloadHandler = null;
		}
	}

	protected onMount(): void {
		if (this.config.startSignal) {
			this.sendSignal(this.config.startSignal);
		}
	}

	protected onDestroy(): void {
		if (this.config.stopSignal) {
			this.sendSignal(this.config.stopSignal);
		}
	}

	protected async sendSignal(signal: string): Promise<void> {
		if (!this.workflowId) return;

		try {
			await pb.send('/api/compliance/send-temporal-signal', {
				method: 'POST',
				body: {
					workflow_id: this.workflowId,
					namespace: this.namespace,
					signal: signal
				}
			});
		} catch (err) {
			console.error(`Error sending signal ${signal}:`, err);
		}
	}

	getWorkflowLogsProps(): WorkflowLogsProps | null {
		if (!this.workflowId || !this.namespace || !this.config.logTransformer) {
			return null;
		}

		return {
			subscriptionSuffix: this.config.subscriptionSuffix as 'openidnet-logs' | 'eudiw-logs',
			startSignal: this.config.startSignal!,
			stopSignal: this.config.stopSignal!,
			workflowSignalSuffix: this.config.workflowSignalSuffix,
			workflowId: this.workflowId,
			namespace: this.namespace,
			logTransformer: this.config.logTransformer
		};
	}
}
