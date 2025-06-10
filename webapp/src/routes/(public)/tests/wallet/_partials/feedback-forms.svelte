<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Alert from '@/components/ui-custom/alert.svelte';
	import { FeedbackForms, type FeedbackFormProps } from './feedback-forms.svelte.js';
	import { Form, SubmitButton } from '@/forms/index.js';
	import { MediaQuery } from 'svelte/reactivity';
	import { Label } from '@/components/ui/label/index.js';
	import { Separator } from '@/components/ui/separator/index.js';
	import { TextareaField } from '@/forms/fields/index.js';

	//

	let props: FeedbackFormProps = $props();
	const forms = new FeedbackForms(props);

	//

	const sm = new MediaQuery('min-width: 640px');
</script>

{#if forms.status == 'fresh'}
	<div class="ml-16 flex flex-col gap-8 sm:flex-row">
		<Form form={forms.successForm} hide={['submit_button']} class="grow basis-1">
			<div class="space-y-2">
				<Label for="success">If the test succeeded:</Label>
				<SubmitButton id="success" class="w-full bg-green-600 hover:bg-green-700">
					Confirm test success
				</SubmitButton>
			</div>
		</Form>

		<Separator orientation={sm.current ? 'vertical' : 'horizontal'} />

		<Form
			form={forms.failureForm}
			hide={['submit_button']}
			hideRequiredIndicator
			class="space-y-2"
		>
			<TextareaField
				form={forms.failureForm}
				name="reason"
				options={{
					label: 'If something went wrong, please tell us what:'
				}}
			/>
			<SubmitButton class="w-full bg-red-600 hover:bg-red-700">Notify issue</SubmitButton>
		</Form>
	</div>
{:else if forms.status == 'success'}
	<Alert variant="info">Your response was submitted! Thanks :)</Alert>
{:else if forms.status == 'already_answered'}
	<Alert variant="info">This test was already confirmed</Alert>
{/if}
