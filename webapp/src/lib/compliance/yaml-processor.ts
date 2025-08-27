// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { fetchWorkflow, fetchWorkflowHistory, getWorkflowMemo } from '$lib/workflows';

// Protocol and version values (hardcoded based on user requirements)
const PROTOCOL = 'openid4vp_wallet';
const VERSION = 'draft-24';

export interface ComplianceTestResult {
	success: boolean;
	credentialOffer?: string;
	error?: string;
}

/**
 * Processes YAML configuration and extracts credential offer from compliance test
 * @param yamlContent - The YAML configuration string
 * @returns Promise<ComplianceTestResult> - Result containing success status and credential offer if found
 */
export async function processYamlAndExtractCredentialOffer(yamlContent: string): Promise<ComplianceTestResult> {
	if (!yamlContent.trim()) {
		return {
			success: false,
			error: 'YAML configuration is required'
		};
	}

	try {
		const response = await pb.send(
			`/api/compliance/${PROTOCOL}/${VERSION}/save-variables-and-start`,
			{
				method: 'POST',
				body: {
					configs_with_fields: {},
					configs_with_json: {},
					custom_checks: {
						credential_test: {
							form: { test: 'one' },
							yaml: yamlContent
						}
					}
				}
			}
		);

		console.log('Compliance test response:', response);

		if (response.results && response.results.length > 0) {
			const firstResult = response.results[0];
			if (firstResult.WorkflowID && firstResult.WorkflowRunID) {				
				// Poll for workflow completion
				const credentialOffer = await pollForWorkflowCompletion(firstResult.WorkflowID, firstResult.WorkflowRunID);
				return {
					success: true,
					credentialOffer
				};
			}
		}

		return {
			success: true
		};
	} catch (error) {
		console.error('Failed to start compliance test:', error);
		return {
			success: false,
			error: 'Failed to start compliance test'
		};
	}
}

async function getWorkflow(workflowId: string, runId: string) {
	const execution = await fetchWorkflow(workflowId, runId);
	if (execution instanceof Error) return execution;

	const eventHistory = await fetchWorkflowHistory(workflowId, runId);
	if (eventHistory instanceof Error) return eventHistory;

	const memo = getWorkflowMemo(execution);
	if (memo instanceof Error) return memo;

	return {
		execution,
		eventHistory,
		memo
	};
}

async function pollForWorkflowCompletion(workflowId: string, runId: string): Promise<string | undefined> {
	const maxAttempts = 600; // Maximum 5 minutes (600 attempts * 500ms)
	let attempts = 0;

	console.log(`Starting to poll for workflow completion: ${workflowId}/${runId}`);

	while (attempts < maxAttempts) {
		try {
			const workflowData = await getWorkflow(workflowId, runId);
			
			if (workflowData instanceof Error) {
				console.error('Failed to fetch workflow data:', workflowData);
				throw new Error('Failed to fetch workflow status');
			}

			console.log(`Poll attempt ${attempts + 1}: Status = ${workflowData.execution.status}`);

			// Check if workflow is completed
			if (workflowData.execution.status === 'Completed') {
				console.log('Workflow completed successfully!');
				console.log('Final workflow data:', workflowData);
				console.log('Execution object keys:', Object.keys(workflowData.execution));
				console.log('Execution object:', workflowData.execution);
				
				// Try to get the actual workflow result
				try {
					const resultResponse = await pb.send(`/api/compliance/checks/${workflowId}/${runId}/result`, {
						method: 'GET'
					});
					console.log('Workflow result from API:', resultResponse);
					
					// The result should have an Output field with the test results
					if (resultResponse.Output) {
						console.log('Test results:', resultResponse.Output);
						console.log('Number of tests:', resultResponse.Output.length);
						
						// Display detailed test results
						resultResponse.Output.forEach((test: unknown, index: number) => {
							console.log(`Test ${index + 1}:`, test);
						});
						
						// Check for credential offer in the first test's first step
						if (resultResponse.Output.length > 0) {
							const firstTest = resultResponse.Output[0];
							if (firstTest.steps && firstTest.steps.length > 0) {
								const firstStep = firstTest.steps[0];
								if (firstStep.captures && firstStep.captures.credentialOffer) {
									const credentialOffer = firstStep.captures.credentialOffer;
									console.log('Found credential offer:', credentialOffer);
									return credentialOffer;
								}
							}
						}
					} else if (resultResponse.Message) {
						console.log('Workflow message:', resultResponse.Message);
					} else {
						console.log('Workflow completed but no specific output found');
					}
				} catch (resultError) {
					console.log('Could not fetch workflow result via API, trying alternative approach');
					console.error('Result fetch error:', resultError);
					
					// Try to find result data in various places in the workflow data
					const execution = workflowData.execution;
					if ((execution as unknown as Record<string, unknown>).workflowExecutionInfo) {
						console.log('WorkflowExecutionInfo:', (execution as unknown as Record<string, unknown>).workflowExecutionInfo);
					}
					
					// Log all available data for debugging
					console.log('Event history:', workflowData.eventHistory);
					console.log('Event history length:', workflowData.eventHistory?.length);
					
					// Look for completion events that might contain the result
					if (workflowData.eventHistory) {
						const completionEvents = workflowData.eventHistory.filter((event: unknown) => {
							const eventObj = event as Record<string, unknown>;
							return eventObj.eventType?.toString().includes('Completed') || 
								   eventObj.eventType?.toString().includes('WorkflowExecutionCompleted');
						});
						console.log('Completion events:', completionEvents);
						
						// Look for any events with result data
						const eventsWithResult = workflowData.eventHistory.filter((event: unknown) => {
							const eventObj = event as Record<string, unknown>;
							const attributes = eventObj.workflowExecutionCompletedEventAttributes as Record<string, unknown> | undefined;
							return attributes?.result ||
								   eventObj.result ||
								   JSON.stringify(event).includes('result');
						});
						console.log('Events with result data:', eventsWithResult);
					}
					
					console.log('Memo:', workflowData.memo);
				}
				return undefined;
			} else if (workflowData.execution.status === 'Failed' || 
					   workflowData.execution.status === 'Terminated' || 
					   workflowData.execution.status === 'Canceled') {
				console.log(`Workflow ${workflowData.execution.status.toLowerCase()}`);
				console.log('Failed execution object:', workflowData.execution);
				throw new Error(`Workflow ${workflowData.execution.status.toLowerCase()}`);
			}

			// Workflow is still running, wait before polling again
			console.log('Workflow still running, waiting 500ms...');
			await new Promise(resolve => setTimeout(resolve, 500));
			attempts++;
		} catch (error) {
			console.error('Error polling workflow:', error);
			throw new Error('Error while waiting for workflow completion');
		}
	}

	// Timeout reached
	console.log('Timeout reached while waiting for workflow completion');
	throw new Error('Workflow did not complete within the expected time (5 minutes)');
}
