<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { getPath } from '$lib/utils';
	import { CalendarIcon } from 'lucide-svelte';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import { Label } from '@/components/ui/label';
	import { Field, SelectField, SelectFieldAny } from '@/forms/fields';
	import Form from '@/forms/form.svelte';
	import { m } from '@/i18n';

	import { createSchedulePipelineForm } from './schedule';
	import { dayOptions, scheduleModeOptions } from './schedule.utils';

	//

	type Props = {
		pipeline: PipelinesResponse;
	};

	let { pipeline }: Props = $props();

	let isOpen = $state(false);

	const form = createSchedulePipelineForm(getPath(pipeline), () => {
		// isOpen = false;
	});

	const formData = form.form;
</script>

<Dialog bind:open={isOpen} title={m.Schedule_workflow()}>
	{#snippet trigger({ props })}
		<Button {...props} size="sm" variant="ghost">
			<CalendarIcon />
			{m.Schedule()}
		</Button>
	{/snippet}

	{#snippet content()}
		<Form {form}>
			<div class="space-y-2">
				<Label>{m.Workflow()}</Label>
				<div class="rounded-md bg-slate-100 p-2">
					{pipeline.name}
				</div>
			</div>

			<SelectField
				{form}
				name="schedule_mode.mode"
				options={{
					items: scheduleModeOptions,
					label: m.interval()
				}}
			/>

			{#if $formData.schedule_mode.mode === 'weekly'}
				<SelectFieldAny
					{form}
					name="schedule_mode.day"
					options={{
						items: dayOptions,
						placeholder: m.Select_a_weekday(),
						label: m.weekday()
					}}
				/>
			{:else if $formData.schedule_mode.mode === 'monthly'}
				<Field
					{form}
					name="schedule_mode.day"
					options={{ type: 'number', label: m.input_a_day() }}
				/>
			{/if}

			{#snippet submitButtonContent()}
				{m.Schedule()}
			{/snippet}
		</Form>
	{/snippet}
</Dialog>
