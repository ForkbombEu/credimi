// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck

import type { HistoryEvent } from '@forkbombeu/temporal-ui';

export const workflowResponse = {
	executionConfig: {
		taskQueue: {
			name: 'openid-test-task-queue',
			kind: 'TASK_QUEUE_KIND_NORMAL'
		},
		workflowExecutionTimeout: '0s',
		workflowRunTimeout: '0s',
		defaultWorkflowTaskTimeout: '10s'
	},
	workflowExecutionInfo: {
		execution: {
			workflowId: 'OpenIDTestWorkflow498249c7-c062-413f-932b-b95bb1cb9ea9',
			runId: '5a09c28f-dfd3-4a86-8dc2-a8a2a4f031cd'
		},
		type: {
			name: 'OpenIDTestWorkflow'
		},
		startTime: '2025-03-13T15:54:23.223594Z',
		closeTime: '2025-03-13T15:55:44.887031Z',
		status: 'WORKFLOW_EXECUTION_STATUS_FAILED',
		historyLength: '23',
		executionTime: '2025-03-13T15:54:23.223594Z',
		memo: {},
		searchAttributes: {
			indexedFields: {
				BuildIds: {
					metadata: {
						encoding: 'anNvbi9wbGFpbg==',
						type: 'S2V5d29yZExpc3Q='
					},
					data: 'WyJ1bnZlcnNpb25lZCIsInVudmVyc2lvbmVkOmI1MmU3ZDJhNGI2ZTE1ZmQwYTI4YmQ5Yjc5ZmU2YzM5Il0='
				}
			}
		},
		autoResetPoints: {
			points: [
				{
					buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39',
					runId: '5a09c28f-dfd3-4a86-8dc2-a8a2a4f031cd',
					firstWorkflowTaskCompletedId: '4',
					createTime: '2025-03-13T15:54:23.271040Z',
					resettable: true
				}
			]
		},
		taskQueue: 'openid-test-task-queue',
		stateTransitionCount: '23',
		historySizeBytes: '7819',
		mostRecentWorkerVersionStamp: {
			buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
		},
		executionDuration: '81.663437s',
		rootExecution: {
			workflowId: 'OpenIDTestWorkflow498249c7-c062-413f-932b-b95bb1cb9ea9',
			runId: '5a09c28f-dfd3-4a86-8dc2-a8a2a4f031cd'
		},
		firstRunId: '5a09c28f-dfd3-4a86-8dc2-a8a2a4f031cd'
	}
};

export const badWorkflowResponse = {
	executionConfig: {
		taskQueue: {
			name: 'OpenIDTestTaskQueue',
			kind: 'TASK_QUEUE_KIND_NORMAL'
		},
		workflowExecutionTimeout: '0s',
		workflowRunTimeout: '0s',
		defaultWorkflowTaskTimeout: '10s'
	},
	workflowExecutionInfo: {
		execution: {
			workflowId: 'OpenIDTestWorkflow8fb36888-be2e-49b1-b6a3-05ed68a28650',
			runId: 'e45c06c4-2db0-4bdf-8249-cfd1995a39bb'
		},
		type: {
			name: 'OpenIDTestWorkflow'
		},
		startTime: '2025-04-07T15:53:02.652055Z',
		status: 'WORKFLOW_EXECUTION_STATUS_RUNNING', // Issue here!
		historyLength: '2',
		executionTime: '2025-04-07T15:53:02.652055Z',
		memo: {},
		searchAttributes: {},
		autoResetPoints: {},
		taskQueue: 'OpenIDTestTaskQueue',
		stateTransitionCount: '1',
		historySizeBytes: '1392',
		rootExecution: {
			workflowId: 'OpenIDTestWorkflow8fb36888-be2e-49b1-b6a3-05ed68a28650',
			runId: 'e45c06c4-2db0-4bdf-8249-cfd1995a39bb'
		},
		firstRunId: 'e45c06c4-2db0-4bdf-8249-cfd1995a39bb'
	},
	pendingWorkflowTask: {
		state: 'PENDING_WORKFLOW_TASK_STATE_SCHEDULED',
		scheduledTime: '2025-04-07T15:53:02.652123Z',
		originalScheduledTime: '2025-04-07T15:53:02.652122Z',
		attempt: 1
	}
};

export const eventHistory: { history: { events: Array<HistoryEvent> } } = {
	history: {
		events: [
			{
				eventId: '1',
				eventTime: '2025-03-13T15:54:23.223594Z',
				eventType: 'EVENT_TYPE_WORKFLOW_EXECUTION_STARTED',
				taskId: '3145737',
				workflowExecutionStartedEventAttributes: {
					workflowType: {
						name: 'OpenIDTestWorkflow'
					},
					taskQueue: {
						name: 'openid-test-task-queue',
						kind: 'TASK_QUEUE_KIND_NORMAL'
					},
					input: {
						payloads: [
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'eyJWYXJpYW50Ijoie1wiY3JlZGVudGlhbF9mb3JtYXRcIjpcInNkX2p3dF92Y1wiLFwiY2xpZW50X2lkX3NjaGVtZVwiOlwiZGlkXCIsXCJyZXF1ZXN0X21ldGhvZFwiOlwicmVxdWVzdF91cmlfc2lnbmVkXCIsXCJyZXNwb25zZV9tb2RlXCI6XCJkaXJlY3RfcG9zdFwifSIsIkZvcm0iOnsiYWxpYXMiOiJURVNUX2Zyb21fcmVzdCIsImRlc2NyaXB0aW9uIjoiVEVTVCBGUk9NIEJBU0ggU0NSSVBUIiwic2VydmVyIjp7ImF1dGhvcml6YXRpb25fZW5kcG9pbnQiOiJvcGVuaWQtdmM6Ly8ifSwiY2xpZW50Ijp7ImNsaWVudF9pZCI6ImRpZDp3ZWI6YXBwLmFsdG1lLmlvOmlzc3VlciIsInByZXNlbnRhdGlvbl9kZWZpbml0aW9uIjp7ImlkIjoidHdvX3NkX2p3dCIsImlucHV0X2Rlc2NyaXB0b3JzIjpbeyJjb25zdHJhaW50cyI6eyJmaWVsZHMiOlt7ImZpbHRlciI6eyJjb25zdCI6InVybjpldS5ldXJvcGEuZWMuZXVkaTpwaWQ6MSIsInR5cGUiOiJzdHJpbmcifSwicGF0aCI6WyIkLnZjdCJdfV19LCJmb3JtYXQiOnsidmMrc2Qtand0Ijp7ImtiLWp3dF9hbGdfdmFsdWVzIjpbIkVTMjU2IiwiRVMyNTZLIiwiRWREU0EiXSwic2Qtand0X2FsZ192YWx1ZXMiOlsiRVMyNTYiLCJFUzI1NksiLCJFZERTQSJdfX0sImlkIjoicGlkX2NyZWRlbnRpYWwifV19LCJqd2tzIjp7ImtleXMiOlt7ImFsZyI6IkVTMjU2IiwiY3J2IjoiUC0yNTYiLCJkIjoiR1NibzlUcG1HYUxneHhPNlJOeDZRbnZjZnlrUUpTN3ZVVmdUZTh2eTlXMCIsImt0eSI6IkVDIiwieCI6Im01dUtzRTM1dDNzUDdnam1pclVld3VmeDJHdDJuNko3ZlNXNjhhcEIyTG8iLCJ5IjoiLVY1NFRwTUk4UmJwQjQwaGJBb2NJam5hSFg1V1A2TkhqV2tIZmRDU0F5VSJ9XX19fSwiVXNlck1haWwiOiJwaW5AZ21haWwuY29tIiwiQXBwVVJMIjoiaHR0cDovL2xvY2FsaG9zdDo4MDkwIn0='
							}
						]
					},
					workflowExecutionTimeout: '0s',
					workflowRunTimeout: '0s',
					workflowTaskTimeout: '10s',
					originalExecutionRunId: '5a09c28f-dfd3-4a86-8dc2-a8a2a4f031cd',
					identity: '66470@MacBookPro.fritz.box@',
					firstExecutionRunId: '5a09c28f-dfd3-4a86-8dc2-a8a2a4f031cd',
					attempt: 1,
					firstWorkflowTaskBackoff: '0s',
					header: {},
					workflowId: 'OpenIDTestWorkflow498249c7-c062-413f-932b-b95bb1cb9ea9'
				}
			},
			{
				eventId: '2',
				eventTime: '2025-03-13T15:54:23.225539Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_SCHEDULED',
				taskId: '3145738',
				workflowTaskScheduledEventAttributes: {
					taskQueue: {
						name: 'openid-test-task-queue',
						kind: 'TASK_QUEUE_KIND_NORMAL'
					},
					startToCloseTimeout: '10s',
					attempt: 1
				}
			},
			{
				eventId: '3',
				eventTime: '2025-03-13T15:54:23.241081Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_STARTED',
				taskId: '3145743',
				workflowTaskStartedEventAttributes: {
					scheduledEventId: '2',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: '5142515b-a0d5-439e-8645-e45b85608184',
					historySizeBytes: '1274',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '4',
				eventTime: '2025-03-13T15:54:23.271039Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_COMPLETED',
				taskId: '3145747',
				workflowTaskCompletedEventAttributes: {
					scheduledEventId: '2',
					startedEventId: '3',
					identity: '66465@MacBookPro.fritz.box@',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					},
					sdkMetadata: {
						langUsedFlags: [3],
						sdkName: 'temporal-go',
						sdkVersion: '1.31.0'
					},
					meteringMetadata: {}
				}
			},
			{
				eventId: '5',
				eventTime: '2025-03-13T15:54:23.271786Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_SCHEDULED',
				taskId: '3145748',
				activityTaskScheduledEventAttributes: {
					activityId: '5',
					activityType: {
						name: 'GenerateYAMLActivity'
					},
					taskQueue: {
						name: 'openid-test-task-queue',
						kind: 'TASK_QUEUE_KIND_NORMAL'
					},
					header: {},
					input: {
						payloads: [
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'IntcImNyZWRlbnRpYWxfZm9ybWF0XCI6XCJzZF9qd3RfdmNcIixcImNsaWVudF9pZF9zY2hlbWVcIjpcImRpZFwiLFwicmVxdWVzdF9tZXRob2RcIjpcInJlcXVlc3RfdXJpX3NpZ25lZFwiLFwicmVzcG9uc2VfbW9kZVwiOlwiZGlyZWN0X3Bvc3RcIn0i'
							},
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'eyJhbGlhcyI6IlRFU1RfZnJvbV9yZXN0IiwiZGVzY3JpcHRpb24iOiJURVNUIEZST00gQkFTSCBTQ1JJUFQiLCJzZXJ2ZXIiOnsiYXV0aG9yaXphdGlvbl9lbmRwb2ludCI6Im9wZW5pZC12YzovLyJ9LCJjbGllbnQiOnsiY2xpZW50X2lkIjoiZGlkOndlYjphcHAuYWx0bWUuaW86aXNzdWVyIiwicHJlc2VudGF0aW9uX2RlZmluaXRpb24iOnsiaWQiOiJ0d29fc2Rfand0IiwiaW5wdXRfZGVzY3JpcHRvcnMiOlt7ImNvbnN0cmFpbnRzIjp7ImZpZWxkcyI6W3siZmlsdGVyIjp7ImNvbnN0IjoidXJuOmV1LmV1cm9wYS5lYy5ldWRpOnBpZDoxIiwidHlwZSI6InN0cmluZyJ9LCJwYXRoIjpbIiQudmN0Il19XX0sImZvcm1hdCI6eyJ2YytzZC1qd3QiOnsia2Itand0X2FsZ192YWx1ZXMiOlsiRVMyNTYiLCJFUzI1NksiLCJFZERTQSJdLCJzZC1qd3RfYWxnX3ZhbHVlcyI6WyJFUzI1NiIsIkVTMjU2SyIsIkVkRFNBIl19fSwiaWQiOiJwaWRfY3JlZGVudGlhbCJ9XX0sImp3a3MiOnsia2V5cyI6W3siYWxnIjoiRVMyNTYiLCJjcnYiOiJQLTI1NiIsImQiOiJHU2JvOVRwbUdhTGd4eE82Uk54NlFudmNmeWtRSlM3dlVWZ1RlOHZ5OVcwIiwia3R5IjoiRUMiLCJ4IjoibTV1S3NFMzV0M3NQN2dqbWlyVWV3dWZ4Mkd0Mm42SjdmU1c2OGFwQjJMbyIsInkiOiItVjU0VHBNSThSYnBCNDBoYkFvY0lqbmFIWDVXUDZOSGpXa0hmZENTQXlVIn1dfX19'
							},
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'Ii92YXIvZm9sZGVycy81bC8zNHJtOGNfczNqOWZuM2tmaGY4YmpiMTQwMDAwZ24vVC9nZW5lcmF0ZWQtODIxODcxMjI1LnlhbWwi'
							}
						]
					},
					scheduleToCloseTimeout: '600s',
					scheduleToStartTimeout: '600s',
					startToCloseTimeout: '300s',
					heartbeatTimeout: '0s',
					workflowTaskCompletedEventId: '4',
					retryPolicy: {
						initialInterval: '5s',
						backoffCoefficient: 2,
						maximumInterval: '60s',
						maximumAttempts: 5
					},
					useWorkflowBuildId: true
				}
			},
			{
				eventId: '6',
				eventTime: '2025-03-13T15:54:23.274925Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_STARTED',
				taskId: '3145755',
				activityTaskStartedEventAttributes: {
					scheduledEventId: '5',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: 'c05a3fd9-4994-416c-a4fd-ac9c782496f4',
					attempt: 1,
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '7',
				eventTime: '2025-03-13T15:54:23.282032Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_COMPLETED',
				taskId: '3145756',
				activityTaskCompletedEventAttributes: {
					scheduledEventId: '5',
					startedEventId: '6',
					identity: '66465@MacBookPro.fritz.box@'
				}
			},
			{
				eventId: '8',
				eventTime: '2025-03-13T15:54:23.282059Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_SCHEDULED',
				taskId: '3145757',
				workflowTaskScheduledEventAttributes: {
					taskQueue: {
						name: 'MacBookPro.fritz.box:46a0cddf-afab-4457-b7f9-a7d53218d433',
						kind: 'TASK_QUEUE_KIND_STICKY',
						normalName: 'openid-test-task-queue'
					},
					startToCloseTimeout: '10s',
					attempt: 1
				}
			},
			{
				eventId: '9',
				eventTime: '2025-03-13T15:54:23.284339Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_STARTED',
				taskId: '3145761',
				workflowTaskStartedEventAttributes: {
					scheduledEventId: '8',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: 'd5011359-f66c-4386-9827-184ef812a59e',
					historySizeBytes: '2987',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '10',
				eventTime: '2025-03-13T15:54:23.288366Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_COMPLETED',
				taskId: '3145765',
				workflowTaskCompletedEventAttributes: {
					scheduledEventId: '8',
					startedEventId: '9',
					identity: '66465@MacBookPro.fritz.box@',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					},
					sdkMetadata: {},
					meteringMetadata: {}
				}
			},
			{
				eventId: '11',
				eventTime: '2025-03-13T15:54:23.288396Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_SCHEDULED',
				taskId: '3145766',
				activityTaskScheduledEventAttributes: {
					activityId: '11',
					activityType: {
						name: 'RunStepCIJSProgramActivity'
					},
					taskQueue: {
						name: 'openid-test-task-queue',
						kind: 'TASK_QUEUE_KIND_NORMAL'
					},
					header: {},
					input: {
						payloads: [
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'Ii92YXIvZm9sZGVycy81bC8zNHJtOGNfczNqOWZuM2tmaGY4YmpiMTQwMDAwZ24vVC9nZW5lcmF0ZWQtODIxODcxMjI1LnlhbWwi'
							},
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'Ild6N1NNazNHNk1VUldlSkdnT3dWTXp6RnMzdTZ3WEFOUkk4QTgrOTZLZnJyRVRJRCtGWWQydEorSzFzYjVyaC9FL3h6b29sS1c2bGhoK0lDcUFOLzlBPT0i'
							}
						]
					},
					scheduleToCloseTimeout: '600s',
					scheduleToStartTimeout: '600s',
					startToCloseTimeout: '300s',
					heartbeatTimeout: '0s',
					workflowTaskCompletedEventId: '10',
					retryPolicy: {
						initialInterval: '5s',
						backoffCoefficient: 2,
						maximumInterval: '60s',
						maximumAttempts: 5
					},
					useWorkflowBuildId: true
				}
			},
			{
				eventId: '12',
				eventTime: '2025-03-13T15:54:23.291047Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_STARTED',
				taskId: '3145772',
				activityTaskStartedEventAttributes: {
					scheduledEventId: '11',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: 'eb4de3a5-31e4-44eb-a987-950d28edbea1',
					attempt: 1,
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '13',
				eventTime: '2025-03-13T15:54:29.820410Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_COMPLETED',
				taskId: '3145773',
				activityTaskCompletedEventAttributes: {
					result: {
						payloads: [
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'eyJpZCI6InFYS0lIYkxBeE9tY201NCIsInBsYW5faWQiOiI5Zk1ubGMxRHBNZ2dtIiwicmVzdWx0Ijoib3BlbmlkLXZjOi8vP3JlcXVlc3RfdXJpPWh0dHBzOi8vd3d3LmNlcnRpZmljYXRpb24ub3BlbmlkLm5ldC90ZXN0L2EvVEVTVF9mcm9tX3Jlc3QvcmVxdWVzdHVyaS9FT1RVQVFsT1pZWG1qd0doSTE0YXRCUkY4eDJQUXprUHRQcW1Lc1BMbDQ5ZmhsZ0pmOWZlZGR0SWI2b1NoMVR1JTIzdDhJRFpvRURTa1IwanJEcnZ2ZkhXQlFHOGtpWDU5dnpjS1ktYTBUSjlSSVx1MDAyNmNsaWVudF9pZD1kaWQ6d2ViOmFwcC5hbHRtZS5pbzppc3N1ZXIifQ=='
							}
						]
					},
					scheduledEventId: '11',
					startedEventId: '12',
					identity: '66465@MacBookPro.fritz.box@'
				}
			},
			{
				eventId: '14',
				eventTime: '2025-03-13T15:54:29.820415Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_SCHEDULED',
				taskId: '3145774',
				workflowTaskScheduledEventAttributes: {
					taskQueue: {
						name: 'MacBookPro.fritz.box:46a0cddf-afab-4457-b7f9-a7d53218d433',
						kind: 'TASK_QUEUE_KIND_STICKY',
						normalName: 'openid-test-task-queue'
					},
					startToCloseTimeout: '10s',
					attempt: 1
				}
			},
			{
				eventId: '15',
				eventTime: '2025-03-13T15:54:29.823759Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_STARTED',
				taskId: '3145778',
				workflowTaskStartedEventAttributes: {
					scheduledEventId: '14',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: '487bbe48-197c-442e-9e3e-e24cb8b58ade',
					historySizeBytes: '4244',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '16',
				eventTime: '2025-03-13T15:54:29.827048Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_COMPLETED',
				taskId: '3145782',
				workflowTaskCompletedEventAttributes: {
					scheduledEventId: '14',
					startedEventId: '15',
					identity: '66465@MacBookPro.fritz.box@',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					},
					sdkMetadata: {},
					meteringMetadata: {}
				}
			},
			{
				eventId: '17',
				eventTime: '2025-03-13T15:54:29.827074Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_SCHEDULED',
				taskId: '3145783',
				activityTaskScheduledEventAttributes: {
					activityId: '17',
					activityType: {
						name: 'SendMailActivity'
					},
					taskQueue: {
						name: 'openid-test-task-queue',
						kind: 'TASK_QUEUE_KIND_NORMAL'
					},
					header: {},
					input: {
						payloads: [
							{
								metadata: {
									encoding: 'anNvbi9wbGFpbg=='
								},
								data: 'eyJTTVRQSG9zdCI6ImxvY2FsaG9zdCIsIlNNVFBQb3J0IjoxMDI1LCJVc2VybmFtZSI6IiIsIlBhc3N3b3JkIjoiIiwiU2VuZGVyRW1haWwiOiJhZG1pbkBleGFtcGxlLm9yZyIsIlJlY2VpdmVyRW1haWwiOiJwaW5AZ21haWwuY29tIiwiU3ViamVjdCI6IlRlc3QgUVIgQ29kZSBFbWFpbCIsIkJvZHkiOiJcblx0XHRcdTAwM2NodG1sXHUwMDNlXG5cdFx0XHRcdTAwM2Nib2R5XHUwMDNlXG5cdFx0XHRcdFx1MDAzY3BcdTAwM2VIZXJlIGlzIHlvdXIgbGluazpcdTAwM2MvcFx1MDAzZVxuXHRcdFx0XHRcdTAwM2NwXHUwMDNlXHUwMDNjYSBocmVmPVwiaHR0cDovL2xvY2FsaG9zdDo4MDkwL3Rlc3RzL3dhbGxldD9xcj1vcGVuaWQtdmMlM0ElMkYlMkYlM0ZyZXF1ZXN0X3VyaSUzRGh0dHBzJTNBJTJGJTJGd3d3LmNlcnRpZmljYXRpb24ub3BlbmlkLm5ldCUyRnRlc3QlMkZhJTJGVEVTVF9mcm9tX3Jlc3QlMkZyZXF1ZXN0dXJpJTJGRU9UVUFRbE9aWVhtandHaEkxNGF0QlJGOHgyUFF6a1B0UHFtS3NQTGw0OWZobGdKZjlmZWRkdEliNm9TaDFUdSUyNTIzdDhJRFpvRURTa1IwanJEcnZ2ZkhXQlFHOGtpWDU5dnpjS1ktYTBUSjlSSSUyNmNsaWVudF9pZCUzRGRpZCUzQXdlYiUzQWFwcC5hbHRtZS5pbyUzQWlzc3Vlclx1MDAyNndvcmtmbG93LWlkPU9wZW5JRFRlc3RXb3JrZmxvdzQ5ODI0OWM3LWMwNjItNDEzZi05MzJiLWI5NWJiMWNiOWVhOVwiIHRhcmdldD1cIl9ibGFua1wiIHJlbD1cIm5vb3BlbmVyXCJcdTAwM2VodHRwOi8vbG9jYWxob3N0OjgwOTAvdGVzdHMvd2FsbGV0P3FyPW9wZW5pZC12YyUzQSUyRiUyRiUzRnJlcXVlc3RfdXJpJTNEaHR0cHMlM0ElMkYlMkZ3d3cuY2VydGlmaWNhdGlvbi5vcGVuaWQubmV0JTJGdGVzdCUyRmElMkZURVNUX2Zyb21fcmVzdCUyRnJlcXVlc3R1cmklMkZFT1RVQVFsT1pZWG1qd0doSTE0YXRCUkY4eDJQUXprUHRQcW1Lc1BMbDQ5ZmhsZ0pmOWZlZGR0SWI2b1NoMVR1JTI1MjN0OElEWm9FRFNrUjBqckRydnZmSFdCUUc4a2lYNTl2emNLWS1hMFRKOVJJJTI2Y2xpZW50X2lkJTNEZGlkJTNBd2ViJTNBYXBwLmFsdG1lLmlvJTNBaXNzdWVyXHUwMDI2d29ya2Zsb3ctaWQ9T3BlbklEVGVzdFdvcmtmbG93NDk4MjQ5YzctYzA2Mi00MTNmLTkzMmItYjk1YmIxY2I5ZWE5XHUwMDNjL2FcdTAwM2VcdTAwM2MvcFx1MDAzZVxuXHRcdFx0XHUwMDNjL2JvZHlcdTAwM2Vcblx0XHRcdTAwM2MvaHRtbFx1MDAzZVxuXHQiLCJBdHRhY2htZW50cyI6bnVsbH0='
							}
						]
					},
					scheduleToCloseTimeout: '600s',
					scheduleToStartTimeout: '600s',
					startToCloseTimeout: '300s',
					heartbeatTimeout: '0s',
					workflowTaskCompletedEventId: '16',
					retryPolicy: {
						initialInterval: '5s',
						backoffCoefficient: 2,
						maximumInterval: '60s',
						maximumAttempts: 5
					},
					useWorkflowBuildId: true
				}
			},
			{
				eventId: '18',
				eventTime: '2025-03-13T15:55:44.872801Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_STARTED',
				taskId: '3145805',
				activityTaskStartedEventAttributes: {
					scheduledEventId: '17',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: '39d5795c-91af-45aa-8dca-9bef9a7cda01',
					attempt: 5,
					lastFailure: {
						message:
							'failed to send email: dial tcp [::1]:1025: connect: connection refused',
						source: 'GoSDK',
						cause: {
							message: 'dial tcp [::1]:1025: connect: connection refused',
							source: 'GoSDK',
							cause: {
								message: 'connect: connection refused',
								source: 'GoSDK',
								cause: {
									message: 'connection refused',
									source: 'GoSDK',
									applicationFailureInfo: {
										type: 'Errno'
									}
								},
								applicationFailureInfo: {
									type: 'SyscallError'
								}
							},
							applicationFailureInfo: {
								type: 'OpError'
							}
						},
						applicationFailureInfo: {
							type: 'wrapError'
						}
					},
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '19',
				eventTime: '2025-03-13T15:55:44.880385Z',
				eventType: 'EVENT_TYPE_ACTIVITY_TASK_FAILED',
				taskId: '3145806',
				activityTaskFailedEventAttributes: {
					failure: {
						message:
							'failed to send email: dial tcp [::1]:1025: connect: connection refused',
						source: 'GoSDK',
						cause: {
							message: 'dial tcp [::1]:1025: connect: connection refused',
							source: 'GoSDK',
							cause: {
								message: 'connect: connection refused',
								source: 'GoSDK',
								cause: {
									message: 'connection refused',
									source: 'GoSDK',
									applicationFailureInfo: {
										type: 'Errno'
									}
								},
								applicationFailureInfo: {
									type: 'SyscallError'
								}
							},
							applicationFailureInfo: {
								type: 'OpError'
							}
						},
						applicationFailureInfo: {
							type: 'wrapError'
						}
					},
					scheduledEventId: '17',
					startedEventId: '18',
					identity: '66465@MacBookPro.fritz.box@',
					retryState: 'RETRY_STATE_MAXIMUM_ATTEMPTS_REACHED'
				}
			},
			{
				eventId: '20',
				eventTime: '2025-03-13T15:55:44.880392Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_SCHEDULED',
				taskId: '3145807',
				workflowTaskScheduledEventAttributes: {
					taskQueue: {
						name: 'MacBookPro.fritz.box:46a0cddf-afab-4457-b7f9-a7d53218d433',
						kind: 'TASK_QUEUE_KIND_STICKY',
						normalName: 'openid-test-task-queue'
					},
					startToCloseTimeout: '10s',
					attempt: 1
				}
			},
			{
				eventId: '21',
				eventTime: '2025-03-13T15:55:44.882815Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_STARTED',
				taskId: '3145811',
				workflowTaskStartedEventAttributes: {
					scheduledEventId: '20',
					identity: '66465@MacBookPro.fritz.box@',
					requestId: 'ca41485d-336c-444e-aad7-76541bce7329',
					historySizeBytes: '6714',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					}
				}
			},
			{
				eventId: '22',
				eventTime: '2025-03-13T15:55:44.886956Z',
				eventType: 'EVENT_TYPE_WORKFLOW_TASK_COMPLETED',
				taskId: '3145815',
				workflowTaskCompletedEventAttributes: {
					scheduledEventId: '20',
					startedEventId: '21',
					identity: '66465@MacBookPro.fritz.box@',
					workerVersion: {
						buildId: 'b52e7d2a4b6e15fd0a28bd9b79fe6c39'
					},
					sdkMetadata: {},
					meteringMetadata: {}
				}
			},
			{
				eventId: '23',
				eventTime: '2025-03-13T15:55:44.887031Z',
				eventType: 'EVENT_TYPE_WORKFLOW_EXECUTION_FAILED',
				taskId: '3145816',
				workflowExecutionFailedEventAttributes: {
					failure: {
						message:
							'failed to print QR code to terminal: activity error (type: SendMailActivity, scheduledEventID: 17, startedEventID: 18, identity: 66465@MacBookPro.fritz.box@): failed to send email: dial tcp [::1]:1025: connect: connection refused (type: wrapError, retryable: true): dial tcp [::1]:1025: connect: connection refused (type: OpError, retryable: true): connect: connection refused (type: SyscallError, retryable: true): connection refused (type: Errno, retryable: true)',
						source: 'GoSDK',
						cause: {
							message: 'activity error',
							source: 'GoSDK',
							cause: {
								message:
									'failed to send email: dial tcp [::1]:1025: connect: connection refused',
								source: 'GoSDK',
								cause: {
									message: 'dial tcp [::1]:1025: connect: connection refused',
									source: 'GoSDK',
									cause: {
										message: 'connect: connection refused',
										source: 'GoSDK',
										cause: {
											message: 'connection refused',
											source: 'GoSDK',
											applicationFailureInfo: {
												type: 'Errno'
											}
										},
										applicationFailureInfo: {
											type: 'SyscallError'
										}
									},
									applicationFailureInfo: {
										type: 'OpError'
									}
								},
								applicationFailureInfo: {
									type: 'wrapError'
								}
							},
							activityFailureInfo: {
								scheduledEventId: '17',
								startedEventId: '18',
								identity: '66465@MacBookPro.fritz.box@',
								activityType: {
									name: 'SendMailActivity'
								},
								activityId: '17',
								retryState: 'RETRY_STATE_MAXIMUM_ATTEMPTS_REACHED'
							}
						},
						applicationFailureInfo: {
							type: 'wrapError'
						}
					},
					retryState: 'RETRY_STATE_RETRY_POLICY_NOT_SET',
					workflowTaskCompletedEventId: '22'
				}
			}
		]
	}
};
