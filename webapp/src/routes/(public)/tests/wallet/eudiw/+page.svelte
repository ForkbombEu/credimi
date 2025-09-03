<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import { getEUDIWWorkflowLogsProps } from '$lib/wallet-test-pages/eudiw';

	import Alert from '@/components/ui-custom/alert.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import FeedbackForms from '../_partials/feedback-forms.svelte';
	import QrLink from '../_partials/qr-link.svelte';
	import Step from '../_partials/step.svelte';
	import WorkflowLogs from '../_partials/workflow-logs.svelte';

	//

	let { data } = $props();
	const { qr, workflowId, namespace } = $derived(data);

	const workflowLogsProps = $derived(getEUDIWWorkflowLogsProps(workflowId, namespace));
</script>

<PageContent>
	<T tag="h1" class="mb-4">{m.EUDIW_Wallet_test()}</T>
	<div class="space-y-4">
		{#if qr}
			<Step n="1" text="Scan this QR with the wallet app to start the check">
				<div
					class="ml-16 mt-4 flex flex-col items-center justify-center rounded-md bg-primary/10 p-2 sm:flex-row"
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
