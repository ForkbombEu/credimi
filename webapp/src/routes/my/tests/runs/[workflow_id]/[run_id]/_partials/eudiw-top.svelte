<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { LogStatus, type WorkflowLogsProps } from '$wallet-test/_partials/workflow-logs';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';
	import { z } from 'zod';
	import Container from './container.svelte';
	import Section from './section.svelte';

	//

	type Props = {
		workflowId: string;
		namespace: string;
		showFeedbackForm?: boolean;
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
