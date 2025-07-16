<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onDestroy } from 'svelte';
	import { 
		createOpenIdNetWorkflowManager, 
		createEudiwWorkflowManager, 
		createEwcWorkflowManager 
	} from './index';

	type WorkflowType = 'openidnet' | 'eudiw' | 'ewc';

	type Props = {
		workflowType: WorkflowType;
		workflowId: string;
		namespace: string;
		runId?: string;
		children?: Snippet;
		containerComponent?: any; // For Container component
		stepComponent?: any; // For Step component  
		feedbackFormsComponent?: any; // For FeedbackForms component
		workflowLogsComponent?: any; // For WorkflowLogs component
		showLogs?: boolean;
		showFeedbackForms?: boolean;
	};

	let { 
		workflowType,
		workflowId, 
		namespace, 
		runId,
		children,
		containerComponent: Container,
		stepComponent: Step,
		feedbackFormsComponent: FeedbackForms,
		workflowLogsComponent: WorkflowLogs,
		showLogs = true,
		showFeedbackForms = true
	}: Props = $props();

	// Create workflow manager instance based on workflow type
	let workflowManager: ReturnType<typeof createOpenIdNetWorkflowManager | typeof createEudiwWorkflowManager | typeof createEwcWorkflowManager> | null = null;

	$effect(() => {
		if (workflowId && namespace) {
			workflowManager?.destroy(); // Clean up previous instance
			
			switch (workflowType) {
				case 'openidnet':
					workflowManager = createOpenIdNetWorkflowManager(workflowId, namespace);
					break;
				case 'eudiw':
					workflowManager = createEudiwWorkflowManager(workflowId, namespace);
					break;
				case 'ewc':
					workflowManager = createEwcWorkflowManager(workflowId, namespace);
					break;
			}
		}
	});

	// Clean up on component destroy
	onDestroy(() => {
		workflowManager?.destroy();
	});

	// Setup beforeunload cleanup for cross-browser compatibility
	function handleBeforeUnload() {
		workflowManager?.destroy();
	}

	const workflowLogsProps = $derived.by(() => {
		return workflowManager?.getWorkflowLogsProps() || null;
	});

	const shouldShowLogs = $derived(showLogs && workflowLogsProps && WorkflowLogs);
	const shouldShowFeedbackForms = $derived(showFeedbackForms && FeedbackForms);
</script>

<svelte:window on:beforeunload={handleBeforeUnload} />

{#if Container}
	<Container>
		{#snippet left()}
			{#if children}
				<div class:space-y-4={shouldShowFeedbackForms}>
					{@render children()}
					{#if shouldShowFeedbackForms && Step}
						<Step text="Confirm the result">
							<FeedbackForms {workflowId} {namespace} class="!gap-4 pt-4" />
						</Step>
					{/if}
				</div>
			{:else if shouldShowFeedbackForms && Step}
				<Step text="Confirm the result">
					<FeedbackForms {workflowId} {namespace} class="!gap-4 pt-4" />
				</Step>
			{/if}
		{/snippet}

		{#if shouldShowLogs}
			{#snippet right()}
				<Step text="Logs" class="h-full">
					<div class="pt-4">
						<WorkflowLogs {...workflowLogsProps} uiSize="sm" class="!max-h-[500px]" />
					</div>
				</Step>
			{/snippet}
		{/if}
	</Container>
{:else}
	<!-- Fallback if no Container component provided -->
	<div class="unified-top-fallback">
		{@render children?.()}
	</div>
{/if}
