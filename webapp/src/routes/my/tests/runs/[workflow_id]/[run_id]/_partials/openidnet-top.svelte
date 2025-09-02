<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { getOpenIDNetWorkflowLogsProps } from '$lib/wallet-test-pages/openidnet';
	import WorkflowLogs from '$wallet-test/_partials/workflow-logs.svelte';

	import Container from './container.svelte';
	import Section from './section.svelte';

	//

	type Props = {
		workflowId: string;
		runId: string;
		namespace: string;
	};

	let { workflowId, namespace }: Props = $props();
	const workflowLogsProps = $derived(getOpenIDNetWorkflowLogsProps(workflowId, namespace));
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
