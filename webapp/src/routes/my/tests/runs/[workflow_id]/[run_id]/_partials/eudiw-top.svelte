<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Step from '$wallet-test/_partials/step.svelte';
	import FeedbackForms from '$wallet-test/_partials/feedback-forms.svelte';
	import { LogStatus, type WorkflowLogsProps } from '$wallet-test/_partials/workflow-logs';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';
	import { z } from 'zod';
	import Container from './container.svelte';

	//

	type Props = {
		workflowId: string;
		namespace: string;
	};

	let { workflowId, namespace }: Props = $props();

	//

	const workflowLogsProps: WorkflowLogsProps = $derived.by(() => {
		if (!workflowId || !namespace) {
			throw new Error('missing workflowId or namespace');
		}
		return {
			subscriptionSuffix: 'eudiw-logs',
			startSignal: 'start-eudiw-check-signal',
			stopSignal: 'stop-eudiw-check-signal',
			workflowId,
			namespace,
			logTransformer: (rawLog) => {
				const data = LogsSchema.parse(rawLog);
				return {
					time: data.timestamp,
					message: data.event + '\n' + data.cause,
					status: LogStatus.INFO,
					rawLog
				};
			}
		};
	});

	const LogsSchema = z
		.object({
			actor: z.string(),
			event: z.string(),
			cause: z.string().optional(),
			timestamp: z.number().optional()
		})
		.passthrough();
</script>

{#if workflowId && namespace}
	<Container>
		{#snippet left()}
			<Step text="Confirm the result">
				<FeedbackForms {workflowId} {namespace} class="!gap-4 pt-4" />
			</Step>
		{/snippet}

		{#snippet right()}
			<Step text="Logs" class="h-full">
				<div class="pt-4">
					<WorkflowLogs {...workflowLogsProps} uiSize="sm" class="!max-h-[500px] " />
				</div>
			</Step>
		{/snippet}
	</Container>
{/if}
