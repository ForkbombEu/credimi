<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import Container from './container.svelte';
	import { WorkflowUICoordinator } from '$lib/qrpages';

	type Props = {
		workflowId: string;
		runId: string;
		namespace: string;
		children?: Snippet;
	};

	let { workflowId, runId, namespace, children }: Props = $props();
</script>

<WorkflowUICoordinator
	workflowType="ewc"
	{workflowId}
	{namespace}
	{children}
	containerComponent={Container}
	showLogs={false}
	showFeedbackForms={false}
/>
