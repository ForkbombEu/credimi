<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FeedbackForms from '$wallet-test/_partials/feedback-forms.svelte';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';
	import { LogStatus, type WorkflowLogsProps } from '$wallet-test/_partials/workflow-logs';
	import { z } from 'zod';
	import Container from './container.svelte';
	import Section from './section.svelte';

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
		<Section title="Logs" bgColor="blue">
			<WorkflowLogs
				{...workflowLogsProps}
				uiSize="sm"
				class="!max-h-[500px]"
				accordionItemClass="rounded-none !border-b !border-gray-500"
				codeClass="!bg-slate-100 rounded-none"
			/>
		</Section>
	{/snippet}
</Container>
