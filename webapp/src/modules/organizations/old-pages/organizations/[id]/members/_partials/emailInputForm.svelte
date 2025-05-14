<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Icon from '@/components/ui-custom/icon.svelte';
	import { createForm, Form, SubmitButton } from '@/forms';
	import { m } from '@/i18n';
	import { readFileAsString, zodFileSchema } from '@/utils/files';
	import { Array as A } from 'effect';
	import { ArrowRight } from 'lucide-svelte';
	import z from 'zod';
	import { zod } from 'sveltekit-superforms/adapters';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { FileField, TextareaField } from '@/forms/fields';
	import Separator from '@/components/ui/separator/separator.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Button from '@/components/ui-custom/button.svelte';

	interface Props {
		onSuccess?: (emails: string[]) => void;
	}

	let { onSuccess = () => {} }: Props = $props();

	/* Input form */

	const acceptedFiles: Record<string, string | string[]> = {
		'Plain text': 'text/plain',
		CSV: 'text/csv',
		Markdown: 'text/markdown',
		XML: ['application/xml', 'text/xml'],
		JSON: 'application/json'
	};

	const acceptedFilesNames = Object.keys(acceptedFiles).join(', ');
	const mimeTypes = Object.values(acceptedFiles).flat();
	const acceptAttribute = mimeTypes.join(', ');

	//

	const schema = z
		.object({
			file_source: zodFileSchema({ mimeTypes }).optional(),
			text_source: z.string().optional()
		})
		.refine((data) => Boolean(data.file_source) || Boolean(data.text_source));

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			onSuccess(await getEmailsFromFormData(form.data));
		},
		options: { dataType: 'form' }
	});
	const { form: formData, tainted } = form;

	//

	let emails: string[] = $state([]);

	async function getEmailsFromFormData(data: typeof $formData) {
		let sources: string[] = [];
		if (data.text_source) sources.push(data.text_source);
		if (data.file_source) sources.push(await readFileAsString(data.file_source));
		return A.dedupe(extractEmailsFromText(sources.join(' ')));
	}

	function extractEmailsFromText(text: string): string[] {
		const emailRegex = /\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b/g;
		const emails = text.match(emailRegex);
		return emails || [];
	}

	function isFormTainted(t: typeof $tainted) {
		return Object.values(t ?? {}).some((s) => s);
	}

	$effect(() => {
		getEmailsFromFormData($formData);
	});

	$effect(() => {
		getEmailsFromFormData($formData).then((res) => (emails = res));
	});
</script>

<!--  -->

<Form {form}>
	<FileField
		{form}
		name="file_source"
		options={{
			accept: acceptAttribute,
			label: m.Upload_a_file_containing_emails(),
			description: `${m.accepted_files()}: ${acceptedFilesNames}`
		}}
	/>

	<div class="flex items-center gap-4">
		<Separator class="w-0 grow" />
		<T>{m.and()} / {m.or()}</T>
		<Separator class="w-0 grow" />
	</div>

	<TextareaField {form} name="text_source" options={{ label: m.Paste_email_addresses_here() }} />

	{#if isFormTainted($tainted) && emails.length == 0}
		<Alert variant="warning">
			{#snippet content({ Title, Description })}
				<Title>{m.Warning()}</Title>
				<Description>
					{m.We_havent_found_any_emails_in_the_provided_documents_please_upload_a_new_file_or_paste_new_content_()}
				</Description>
			{/snippet}
		</Alert>
	{/if}

	{#snippet submitButton()}
		{#if emails.length}
			<Alert variant="success" class="!p-4">
				{#snippet content({ Title })}
					<div class="flex items-center justify-between gap-2">
						<Title
							><span class="mr-1">✅</span> {emails.length} {m.Emails_found()}</Title
						>
						<Button
							onclick={() => {
								onSuccess(emails);
							}}
						>
							<Icon src={ArrowRight} mr></Icon>
							{m.Review_and_confirm()}
						</Button>
					</div>
				{/snippet}
			</Alert>
		{/if}
	{/snippet}
</Form>
