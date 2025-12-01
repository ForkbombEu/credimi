<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script lang="ts">
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import { Label } from '@/components/ui/label';
	import { SelectField } from '@/forms/fields';
	import Form from '@/forms/form.svelte';
	import { m } from '@/i18n';

	import {
		createScheduleWorkflowForm,
		SCHEDULING_INTERVALS,
		schedulingIntervalLabels
	} from './schedule';

	//

	type Props = {
		workflowID: string;
		runID: string;
		workflowName: string;
		isOpen?: boolean;
	};

	let { workflowID, runID, workflowName, isOpen = $bindable(false) }: Props = $props();

	const form = createScheduleWorkflowForm({
		workflowID,
		runID,
		onSuccess: () => {
			isOpen = false;
		}
	});
</script>

<Dialog bind:open={isOpen} title={m.Schedule_workflow()}>
	{#snippet content()}
		<Form {form}>
			<div class="space-y-2">
				<Label>{m.Workflow()}</Label>
				<div class="rounded-md bg-slate-100 p-2">
					{workflowName}
				</div>
			</div>

			<SelectField
				{form}
				name="interval"
				options={{
					items: SCHEDULING_INTERVALS.map((interval) => ({
						label: schedulingIntervalLabels[interval],
						value: interval
					}))
				}}
			/>

			{#snippet submitButtonContent()}
				{m.Schedule()}
			{/snippet}
		</Form>
	{/snippet}
</Dialog>
