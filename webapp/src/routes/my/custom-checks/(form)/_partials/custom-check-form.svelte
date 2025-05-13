<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import { createForm, Form } from '@/forms';
	import {
		SelectField,
		Field,
		FileField,
		TextareaField,
		CodeEditorField,
		CheckboxField
	} from '@/forms/fields';
	import { zod } from 'sveltekit-superforms/adapters';
	import { goto, m } from '@/i18n';
	import { PlusIcon, UploadIcon } from 'lucide-svelte';
	import BackButton from '$lib/layout/back-button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import PageCardSection from '$lib/layout/page-card-section.svelte';
	import { yaml } from '@codemirror/lang-yaml';
	import Button from '@/components/ui-custom/button.svelte';
	import { readFileAsString } from '@/utils/files.js';
	import { parse } from 'yaml';
	import { getExceptionMessage } from '@/utils/errors.js';
	import { currentUser, pb } from '@/pocketbase';
	import { toast } from 'svelte-sonner';
	import type { StandardsWithTestSuites } from '../../../tests/new/_partials/standards-response-schema';
	import type { CustomChecksResponse } from '@/pocketbase/types';
	import _ from 'lodash';

	//

	type Props = {
		standardsAndTestSuites: StandardsWithTestSuites;
		record?: CustomChecksResponse;
	};

	let { standardsAndTestSuites, record }: Props = $props();

	//

	const schema = createCollectionZodSchema('custom_checks').omit({
		organization: true,
		owner: true
	});

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async ({ form: { data } }) => {
			if (formMode === 'new') {
				await pb.collection('custom_checks').create(data);
			} else if (record) {
				await pb.collection('custom_checks').update(record.id, data);
			}
			toast.success(currentLabels.toastMessage);
			await goto('/my/custom-checks');
		},
		options: {
			dataType: 'form'
		},
		initialData: record ? _.omit(record, 'logo') : undefined
	});

	const { errors } = form;

	//

	type FormMode = 'new' | 'edit';

	const formMode = $derived<FormMode>(record ? 'edit' : 'new');

	type FormLabels = {
		title: string;
		submitButton: string;
		toastMessage: string;
	};

	const labels: Record<FormMode, FormLabels> = {
		new: {
			title: m.Add_a_Custom_Conformance_Check(),
			submitButton: m.Add_a_custom_check(),
			toastMessage: m.Custom_check_created()
		},
		edit: {
			title: m.Update_Custom_Conformance_Check(),
			submitButton: m.Update_custom_check(),
			toastMessage: m.Custom_check_updated()
		}
	};

	const currentLabels = $derived(labels[formMode]);

	//

	const standardsOptions = $derived(
		standardsAndTestSuites.flatMap((standard) =>
			standard.versions.map((version) => ({
				value: `${standard.uid}/${version.uid}`,
				label: `${standard.name} – ${version.name}`
			}))
		)
	);

	//

	function startYamlUpload() {
		// Create and trigger a file input
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = '.yaml,.yml';
		input.multiple = false;
		input.onchange = (e) => {
			if (!(e.target instanceof HTMLInputElement)) return;
			const file = e.target.files?.[0];
			if (!(file instanceof File)) return;
			handleYamlUpload(file);
		};
		input.click();
	}

	async function handleYamlUpload(file: File) {
		try {
			const fileContent = await readFileAsString(file);
			parse(fileContent);
			form.form.update((data) => ({
				...data,
				yaml: fileContent
			}));
		} catch (e) {
			$errors['yaml'] = [getExceptionMessage(e)];
		}
	}
</script>

<div class="bg-secondary min-h-screen space-y-10 p-6">
	<BackButton href="/my/custom-checks">
		{m.Back_and_discard()}
	</BackButton>

	<div class="space-y-2 text-center">
		<T tag="h2">
			{currentLabels.title}
		</T>
		<T tag="p">
			{@html m.Custom_check_form_description()}
		</T>
	</div>

	<Form {form}>
		<PageCardSection
			title={m.Standard_and_version()}
			description={m.Standard_and_Version_description()}
		>
			<SelectField
				{form}
				name="standard_and_version"
				options={{
					items: standardsOptions,
					label: m.Compliance_standard(),
					placeholder: m.Select_a_standard_and_version(),
					description: 'e.g. OpenID4VCI Presentation Test – AuthZ Flow'
				}}
			/>
		</PageCardSection>

		<PageCardSection title={m.Check_Metadata()} description={m.Check_metadata_description()}>
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
				<Field {form} name="name" options={{ label: m.Name() }} />

				<div>
					<FileField {form} name="logo" options={{ label: m.Upload_logo() }} />
					<!-- TODO - Improve logo preview -->
					{#if record?.logo}
						<T class="text-xs">Current: {record.logo}</T>
					{/if}
				</div>

				<Field
					{form}
					name="homepage"
					options={{
						type: 'url',
						placeholder: ' ',
						label: m.Homepage(),
						description: `${m.eg()}: https://yourproject.org/`
					}}
				/>
				<Field
					{form}
					name="repository"
					options={{
						type: 'url',
						placeholder: ' ',
						label: m.Repository(),
						description: `${m.eg()}: https://github.com/your-org/custom-checks`
					}}
				/>

				<TextareaField
					{form}
					name="description"
					options={{
						label: m.Description(),
						description: m.Custom_check_description_field()
					}}
				/>
			</div>
		</PageCardSection>

		<PageCardSection
			title={m.YAML_Configuration()}
			description={m.YAML_Configuration_section_description()}
		>
			<CodeEditorField
				{form}
				name="yaml"
				options={{ lang: yaml(), labelRight: yamlFieldLabelRight, minHeight: 200 }}
			/>
		</PageCardSection>

		<PageCardSection title={m.Make_public()} description={m.Make_public_section_description()}>
			<CheckboxField
				{form}
				name="public"
				options={{
					label: `${m.Make_public()}: ${m.other_users_can_run_this_conformanche_check()}`
				}}
			/>
		</PageCardSection>

		{#snippet submitButtonContent()}
			<PlusIcon />
			{currentLabels.submitButton}
		{/snippet}
	</Form>
</div>

{#snippet yamlFieldLabelRight()}
	<Button variant="secondary" onclick={startYamlUpload}>
		<UploadIcon />
		{m.Upload_yaml()}
	</Button>
{/snippet}
