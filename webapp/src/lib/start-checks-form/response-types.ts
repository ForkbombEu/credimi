// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type StartCheckResult = {
	WorkflowId: string;
	WorkflowRunID: string;
	Message: string;
	Errors: null | string;
	Output: null | string;
	Log: null | string;
	Author: string;
};

export type StartChecksResponse = {
	'protocol/version': string;
	message: string;
	results: StartCheckResult[];
};
