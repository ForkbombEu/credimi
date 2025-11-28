<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import { getOpenIDNetWorkflowLogsProps } from '$lib/wallet-test-pages/openidnet';
	import WorkflowLogs from '$lib/workflows/workflow-logs.svelte';

	import Alert from '@/components/ui-custom/alert.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';
	import { QrCode } from '@/qr';

	import FeedbackForms from '../_partials/feedback-forms.svelte';
	import Step from '../_partials/step.svelte';

	//

	let { data } = $props();
	const { qr, workflowId, namespace } = $derived(data);

	const workflowLogsProps = $derived(getOpenIDNetWorkflowLogsProps(workflowId, namespace));
</script>

<PageContent>
	<T tag="h1" class="mb-4">Wallet OpenId test</T>
	<div class="space-y-4">
		{#if qr}
			<Step n="1" text="Scan this QR with the wallet app to start the check">
				<div
					class="bg-primary/10 ml-16 mt-4 flex flex-col items-center justify-center rounded-md p-2 sm:flex-row"
				>
					<QrCode
						src={qr}
						class="size-40 rounded-sm"
						showLink={true}
						linkClass="max-w-sm break-all p-4 font-mono text-xs"
					/>
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
