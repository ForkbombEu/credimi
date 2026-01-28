<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CalendarIcon } from '@lucide/svelte';
	import { getPath } from '$lib/utils';
	import { toast } from 'svelte-sonner';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod/v3';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import Dialog from '@/components/ui-custom/dialog.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { Label } from '@/components/ui/label';
	import { createForm } from '@/forms';
	import { Field, SelectField, SelectFieldAny } from '@/forms/fields';
	import Form from '@/forms/form.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import SelectRunner from '@/components/select-runner.svelte';
	import { SelectRunnerForm } from '@/components/select-runner.svelte.js';

	import { dayOptions, scheduleModeOptions, scheduleModeSchema } from './types';

	//

	type Props = {
		pipeline: PipelinesResponse;
	};

	let { pipeline }: Props = $props();

	//

	let isOpen = $state(false);
	let runnerForm = $state(new SelectRunnerForm());
	let selectedRunnerPath = $state<string | undefined>(undefined);

	const form = createForm({
		adapter: zod(
			z.object({
				pipeline_id: z.string(),
				schedule_mode: scheduleModeSchema,
				global_runner_id: z.string().optional()
			})
		),
		initialData: {
			pipeline_id: getPath(pipeline)
		},
		onSubmit: async ({ form: { data } }) => {
			if (data.schedule_mode.mode === 'monthly') {
				data.schedule_mode.day = data.schedule_mode.day - 1;
			}
			
			// Add global_runner_id if a runner is selected
			if (selectedRunnerPath) {
				data.global_runner_id = selectedRunnerPath;
			}

			await pb.send('/api/my/schedules/start', {
				method: 'POST',
				body: data
			});

			isOpen = false;
			toast.success(m.Pipeline_scheduled_successfully());
		}
	});

	const formData = form.form;

	// Update runner selection
	runnerForm = new SelectRunnerForm((runner) => {
		selectedRunnerPath = getPath(runner);
	});
</script>

<Dialog bind:open={isOpen} title={m.Schedule_pipeline()}>
	{#snippet trigger({ props })}
		<IconButton {...props} icon={CalendarIcon} tooltip={m.Schedule_pipeline()} />
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

			<div class="space-y-2">
				<Label>{m.Runner()} {m.optional()}</Label>
				<SelectRunner form={runnerForm} showSelected={true} />
			</div>

			{#snippet submitButtonContent()}
				{m.Schedule()}
			{/snippet}
		</Form>
	{/snippet}
</Dialog>
