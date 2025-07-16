<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Step from '$wallet-test/_partials/step.svelte';
	import FeedbackForms from '$wallet-test/_partials/feedback-forms.svelte';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';
	import Container from './container.svelte';
	import { WorkflowUICoordinator } from '$lib/qrpages';

	type Props = {
		workflowId: string;
		namespace: string;
	};

	let { workflowId, namespace }: Props = $props();
</script>

{#if workflowId && namespace}
	<WorkflowUICoordinator
		workflowType="eudiw"
		{workflowId}
		{namespace}
		containerComponent={Container}
		stepComponent={Step}
		feedbackFormsComponent={FeedbackForms}
		workflowLogsComponent={WorkflowLogs}
	/>
{/if}
