<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form, SubmitButton } from '@/forms';
	import { Field, TextareaField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import { pb } from '@/pocketbase';
	import { CollectionField } from '@/collections-components';
	import T from '@/components/ui-custom/t.svelte';

	import t from './test.json';
	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import CodeEditorField from '@/forms/fields/codeEditorField.svelte';

	//

	let workflowStarted = $state<boolean>();

	const form = createForm({
		adapter: zod(
			z.object({
				name: z.string(),
				standard: z.string(),
				json: z.string(),
				email: z.string().email()
			})
		),
		initialData: {
			name: 'Wallet test',
			standard: '9g0blea2dj37ph1',
			email: 'pin@gmail.com',
			json: JSON.stringify(t, null, 4)
		},
		onSubmit: async ({ form }) => {
			const { name, standard, json, email } = form.data;

			// /pkg/internal/pb/workflow.go
			// Result type should be `{started:boolean}`
			const result: { started: boolean } = await pb.send('/api/compliance', {
				method: 'POST',
				body: {
					input: JSON.parse(json),
					user_mail: email,
					workflow_id: standard,
					test_name: name
				}
			});

			workflowStarted = result.started;
		}
	});

	const { form: formData } = form;
</script>

{#if !workflowStarted}
	<Form {form} hideRequiredIndicator>
		<Field {form} name="name" options={{ type: 'text', label: m.Wallet_name() }} />
		<CollectionField
			{form}
			name="standard"
			collection="standards"
			options={{ displayFields: ['name'] }}
		/>

		<CodeEditorField {form} name="json" options={{ lang: 'json', maxHeight: 400 }} />

		<div class="space-y-4 rounded-xl border bg-secondary/30 p-4">
			<Field {form} name="email" options={{ type: 'email' }} />
			<T>{m.We_will_send_the_instructions_for_proceeding_with_the_test_to_this_email()}</T>
		</div>

		{#snippet submitButton()}
			<SubmitButton class="flex w-full">{m.Start_check()}</SubmitButton>
		{/snippet}
	</Form>
{:else}
	<Alert variant="info">
		<T class="mb-2">{m.The_test_has_started_successfully()}</T>
		<T>{m.Next_steps()}</T>
		<ul class="list-inside list-disc pl-1">
			<li>{m.Open_your_email({ email: $formData.email })}</li>
			<li>{m.Follow_the_instructions_to_continue_with_the_compliance_check()}</li>
		</ul>
	</Alert>

	<Button
		class="mt-4 w-full"
		onclick={() => {
			workflowStarted = false;
		}}
	>
		{m.Start_a_new_check()}
	</Button>
{/if}
