// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/* Auto-generated from queries.test.json and edited by hand */

export interface FetchWorkflowsResponse {
	executions: WorkflowExecutionWithChildren[] | null;
}

export interface WorkflowExecutionWithChildren {
	execution: Execution;
	type: WorkflowType;
	startTime: string;
	endTime: string;
	executionTime: string;
	status: string;
	taskQueue: string;
	historyEvents: string;
	historySizeBytes: string;
	searchAttributes: WorkflowSearchAttributes;
	memo: WorkflowMemo;
	rootExecution: WorkflowRootExecution;
	stateTransitionCount: string;
	url: string;
	isRunning: boolean;
	canBeTerminated: boolean;
	displayName: string;
	children?: WorkflowChildren[];
}

interface Execution {
	workflowId: string;
	runId: string;
}

interface WorkflowType {
	name: string;
}

interface WorkflowSearchAttributes {
	indexedFields: WorkflowIndexedFields;
}

interface WorkflowIndexedFields {
	BuildIds: WorkflowBuildIds;
}

interface WorkflowBuildIds {
	data: string;
	metadata: WorkflowMetadata;
}

interface WorkflowMetadata {
	encoding: string;
	type: string;
}

interface WorkflowMemo {
	fields: WorkflowFields;
}

interface WorkflowFields {
	test: WorkflowTest;
	userID?: WorkflowUserId;
	author?: WorkflowAuthor;
	standard?: WorkflowStandard;
}

interface WorkflowTest {
	metadata: WorkflowMetadataSimple;
	data: string;
}

interface WorkflowMetadataSimple {
	encoding: string;
}

interface WorkflowUserId {
	metadata: WorkflowMetadataUser;
	data: string;
}

interface WorkflowMetadataUser {
	encoding: string;
}

interface WorkflowAuthor {
	metadata: WorkflowMetadataAuthor;
	data: string;
}

interface WorkflowMetadataAuthor {
	encoding: string;
}

interface WorkflowStandard {
	metadata: WorkflowMetadataStandard;
	data: string;
}

interface WorkflowMetadataStandard {
	encoding: string;
}

interface WorkflowRootExecution {
	workflowId: string;
	runId: string;
}

export interface WorkflowChildren {
	execution: Execution;
	type: WorkflowType;
	startTime: string;
	endTime: string;
	executionTime: string;
	status: string;
	taskQueue: string;
	historyEvents: string;
	historySizeBytes: string;
	searchAttributes: WorkflowSearchAttributes;
	memo: WorkflowChildrenMemo;
	rootExecution: WorkflowRootExecution;
	stateTransitionCount: string;
	parentExecution: WorkflowParentExecution;
	url: string;
	isRunning: boolean;
	canBeTerminated: boolean;
	displayName: string;
	children?: WorkflowChildren[];
}

interface WorkflowChildrenMemo {
	fields?: WorkflowChildrenFields;
}

interface WorkflowChildrenFields {
	author: WorkflowChildrenAuthor;
	standard: WorkflowChildrenStandard;
	test: WorkflowChildrenTest;
}

interface WorkflowChildrenAuthor {
	metadata: WorkflowChildrenMetadataAuthor;
	data: string;
}

interface WorkflowChildrenMetadataAuthor {
	encoding: string;
}

interface WorkflowChildrenStandard {
	metadata: WorkflowChildrenMetadataStandard;
	data: string;
}

interface WorkflowChildrenMetadataStandard {
	encoding: string;
}

interface WorkflowChildrenTest {
	metadata: WorkflowChildrenMetadataTest;
	data: string;
}

interface WorkflowChildrenMetadataTest {
	encoding: string;
}

interface WorkflowParentExecution {
	workflowId: string;
	runId: string;
}
