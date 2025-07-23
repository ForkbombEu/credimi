// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

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
