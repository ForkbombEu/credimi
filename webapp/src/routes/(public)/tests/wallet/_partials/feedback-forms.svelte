<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Label } from '@/components/ui/label/index.js';
	import { Separator } from '@/components/ui/separator/index.js';
	import { TextareaField } from '@/forms/fields/index.js';
	import { Form, SubmitButton } from '@/forms/index.js';

	import { FeedbackForms, type FeedbackFormProps } from './feedback-forms.svelte.js';

	//

	let props: FeedbackFormProps & { class?: string } = $props();
	const forms = new FeedbackForms(props);
</script>

{#if forms.status == 'fresh'}
	<div class={['flex flex-col gap-8 @sm:flex-row', props.class, '@container']}>
		<Form form={forms.successForm} hide={['submit_button']} class=" grow basis-1">
			<div class="space-y-2">
				<Label for="success">If the test succeeded:</Label>
				<SubmitButton id="success" class="w-full bg-green-600 hover:bg-green-700">
					Confirm test success
				</SubmitButton>
			</div>
		</Form>

		<Separator class="hidden @md:block" orientation="vertical" />
		<Separator class="block @md:hidden" orientation="horizontal" />

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
