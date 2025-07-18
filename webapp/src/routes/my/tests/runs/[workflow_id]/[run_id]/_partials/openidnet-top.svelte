<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Step from '$wallet-test/_partials/step.svelte';
	import FeedbackForms from '$wallet-test/_partials/feedback-forms.svelte';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';
	import type { Snippet } from 'svelte';
	import Container from './container.svelte';
	import { WorkflowUICoordinator } from '$lib/qrpages';

	type Props = {
		workflowId: string;
		runId: string;
		namespace: string;
		children?: Snippet;
	};

	let { workflowId, namespace, children }: Props = $props();
</script>

<WorkflowUICoordinator
	workflowType="openidnet"
	{workflowId}
	{namespace}
	{children}
	containerComponent={Container}
	stepComponent={Step}
	feedbackFormsComponent={FeedbackForms}
	workflowLogsComponent={WorkflowLogs}
/>
