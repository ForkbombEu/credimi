<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowStatus } from '@forkbombeu/temporal-ui/dist/types/workflows';
	import type { ClassValue } from 'svelte/elements';

	import { Code, Hourglass, XIcon } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import ScheduleWorkflowForm from './schedule-workflow-form.svelte';
	import { cancelWorkflow } from './utils';

	//

	type Props = {
		workflow: { workflowId: string; runId: string; status: WorkflowStatus; name: string };

		containerClass?: ClassValue;
	};

	let { workflow, containerClass }: Props = $props();

	//

	let isScheduleWorkflowFormOpen = $state(false);
</script>

<div class={['flex gap-2', containerClass]}>
	<Button
		variant="outline"
		onclick={() =>
			cancelWorkflow({
				workflowId: workflow.workflowId,
				runId: workflow.runId
			})}
		disabled={workflow.status !== 'Running'}
		size="sm"
	>
		<XIcon />
		{m.Terminate()}
	</Button>

	<Button onclick={() => (isScheduleWorkflowFormOpen = true)} variant="outline" size="sm">
		<Hourglass />
		{m.Schedule()}
	</Button>

	<Button disabled variant="outline" size="sm">
		<Code />
		{m.Swagger()}
	</Button>
</div>

<ScheduleWorkflowForm
	workflowID={workflow.workflowId}
	runID={workflow.runId}
	workflowName={workflow.name}
	bind:isOpen={isScheduleWorkflowFormOpen}
/>
