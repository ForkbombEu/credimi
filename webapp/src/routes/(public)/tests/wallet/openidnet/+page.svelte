<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { m } from '@/i18n/index.js';
	import Step from '../_partials/step.svelte';
	import QrLink from '../_partials/qr-link.svelte';
	import FeedbackForms from '../_partials/feedback-forms.svelte';
	import WorkflowLogs from '../_partials/workflow-logs.svelte';
	import { LogStatus, type WorkflowLogsProps } from '../_partials/workflow-logic';
	import { z } from 'zod';

	//

	let { data } = $props();
	const { qr, workflowId, namespace } = $derived(data);

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

<PageContent>
	<T tag="h1" class="mb-4">Wallet OpenId test</T>
	<div class="space-y-4">
		{#if qr}
			<Step n="1" text="Scan this QR with the wallet app to start the check">
				<div
					class="bg-primary/10 ml-16 mt-4 flex flex-col items-center justify-center rounded-md p-2 sm:flex-row"
				>
					<QrLink {qr} />
				</div>
			</Step>
		{:else}
			<Alert variant="destructive">
				<T class="font-bold">{m.Error_check_failed()}</T>
				<T>
					{m.An_error_happened_during_the_check_please_read_the_logs_for_more_information()}
				</T>
			</Alert>
		{/if}

		{#if workflowId && namespace}
			<Step n="2" text="Follow the procedure on the wallet app">
				<div class="ml-16">
					<WorkflowLogs {...workflowLogsProps} />
				</div>
			</Step>

			<Step n="3" text="Confirm the result">
				<FeedbackForms {workflowId} {namespace} />
			</Step>
		{/if}
	</div>
</PageContent>
