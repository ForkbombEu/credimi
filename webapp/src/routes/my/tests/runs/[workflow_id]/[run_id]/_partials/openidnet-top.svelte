<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Step from '$wallet-test/_partials/step.svelte';
	import FeedbackForms from '$wallet-test/_partials/feedback-forms.svelte';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';
	import { LogStatus, type WorkflowLogsProps } from '$wallet-test/_partials/workflow-logs';
	import { z } from 'zod';
	import Container from './container.svelte';

	//

	type Props = {
		workflowId: string;
		runId: string;
		namespace: string;
		showFeedbackForm?: boolean;
	};

	let { workflowId, namespace, showFeedbackForm = true }: Props = $props();

	//

	const workflowLogsProps: WorkflowLogsProps = $derived.by(() => {
		if (!workflowId || !namespace) {
			throw new Error('missing workflowId or namespace');
		}
		return {
			subscriptionSuffix: 'openidnet-logs',
			startSignal: 'start-openidnet-check-log-update',
			stopSignal: 'stop-openidnet-check-log-update',
			workflowSignalSuffix: '-log',
			workflowId,
			namespace,
			logTransformer: (rawLog) => {
				const data = LogsSchema.parse(rawLog);
				return {
					time: data.time,
					message: data.msg,
					status: data.result,
					rawLog
				};
			}
		};
	});

	const LogsSchema = z
		.object({
			_id: z.string(),
			msg: z.string(),
			src: z.string(),
			time: z.number().optional(),
			result: z.nativeEnum(LogStatus).optional()
		})
		.passthrough();
</script>

<Container>
	{#snippet left()}
		{#if showFeedbackForm}
			<div class="space-y-4">
				<Step text="Confirm the result">
					<FeedbackForms {workflowId} {namespace} class="!gap-4 pt-4" />
				</Step>
			</div>
		{/if}
	{/snippet}
	{#snippet right()}
		<Step text="Logs">
			<div class="pt-4">
				<WorkflowLogs {...workflowLogsProps} uiSize="sm" class="!max-h-[500px] " />
			</div>
		</Step>
	{/snippet}
</Container>
