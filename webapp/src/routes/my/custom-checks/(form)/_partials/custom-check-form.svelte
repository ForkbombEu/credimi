<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StandardsWithTestSuites } from '$lib/standards';

	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import PageCardSection from '$lib/layout/page-card-section.svelte';
	import StandardAndVersionField from '$lib/standards/standard-and-version-field.svelte';
	import { jsonStringSchema, stepciYamlSchema } from '$lib/utils';
	import { yaml } from '@codemirror/lang-yaml';
	import { GitBranch, HelpCircle, PlusIcon, UploadIcon } from '@lucide/svelte';
	import { String } from 'effect';
	import { run } from 'json_typegen_wasm';
	import _ from 'lodash';
	import { toast } from 'svelte-sonner';
	import { fromStore } from 'svelte/store';
	import { zod } from 'sveltekit-superforms/adapters';

	import type { CustomChecksRecord, CustomChecksResponse } from '@/pocketbase/types';

	import { removeEmptyValues } from '@/collections-components/form';
	import { mockFile } from '@/collections-components/form/collectionFormSetup';
	import Button from '@/components/ui-custom/button.svelte';
	import LinkExternal from '@/components/ui-custom/linkExternal.svelte';
	import { createForm, Form } from '@/forms';
	import {
		CheckboxField,
		CodeEditorField,
		Field,
		LogoField,
		TextareaField
	} from '@/forms/fields';
	import { goto, m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import { getExceptionMessage } from '@/utils/errors.js';
	import { readFileAsString } from '@/utils/files.js';

	//

	type Props = {
		standardsAndTestSuites: StandardsWithTestSuites;
		record?: CustomChecksResponse;
	};

	let { standardsAndTestSuites, record }: Props = $props();

	//

	const schema = createCollectionZodSchema('custom_checks')
		.omit({
			owner: true,
			input_json_schema: true
		})
		.extend({
			yaml: stepciYamlSchema,
			input_json_sample: jsonStringSchema.optional()
		});

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			const data: Partial<CustomChecksRecord> = removeEmptyValues({ ...form.data });

			const jsonSample = form.data.input_json_sample;
			if (!jsonSample || String.isEmpty(jsonSample)) {
				data.input_json_sample = null;
				data.input_json_schema = null;
			} else {
				data.input_json_schema = generateJsonSchema(jsonSample);
			}

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
		// @ts-expect-error - Slight type mismatch, but it works
		initialData: createInitialData(record)
	});

	function generateJsonSchema(json: string) {
		return run(
			'Root',
			json,
			JSON.stringify({
				output_mode: 'json_schema'
			})
		);
	}

	function createInitialData(record?: CustomChecksResponse) {
		if (!record) return undefined;
		const data = _.omit(record, 'input_json_schema');
		if (record.input_json_sample) {
			data.input_json_sample = JSON.stringify(record.input_json_sample, null, 2);
		}
		if (record.logo) {
			// @ts-expect-error - We need to rewrite the the logo from string to file
			data.logo = mockFile(record.logo, { mimeTypes: ['image/png'] });
		}
		return data;
	}

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

	function startYamlUpload() {
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
			form.form.update((data) => ({
				...data,
				yaml: fileContent
			}));
			form.validate('yaml');
		} catch (e) {
			$errors['yaml'] = [getExceptionMessage(e)];
		}
	}

	// Logo preview
	// TODO - Abstract this to a component: it's useful to preview image changes in forms

	const originalLogoUrl = $derived.by(() => {
		if (!record || !record.logo) return undefined;
		return pb.files.getURL(record, record.logo);
	});

	let formState = fromStore(form.form);

	// Parse selected standard and version for contextual links
	const selectedStandardAndVersion = $derived(() => {
		const value = formState.current.standard_and_version;
		if (!value || typeof value !== 'string') return null;

		const [standardUid, versionUid] = value.split('/');
		if (!standardUid || !versionUid) return null;

		const standard = standardsAndTestSuites.find((s) => s.uid === standardUid);
		if (!standard) return null;

		const version = standard.versions.find((v) => v.uid === versionUid);
		if (!version) return null;

		return { standard, version };
	});
</script>

<FocusPageLayout
	title={currentLabels.title}
	description={m.Custom_check_form_description()}
	backButton={{ title: m.Back_and_discard(), href: '/my/custom-checks' }}
>
	<Form {form}>
		<PageCardSection
			title={m.Standard_and_version()}
			description={m.Standard_and_Version_description()}
		>
			{#snippet headerActions()}
				{@const selection = selectedStandardAndVersion()}
				{#if selection}
					{@const { standard, version } = selection}
					<div class="flex flex-wrap gap-2">
						{#if standard.standard_url}
							<LinkExternal
								href={standard.standard_url}
								text="{standard.name} {m.Standard()}"
								icon={HelpCircle}
								title={m.Learn_about_standard({ name: standard.name })}
							/>
						{/if}

						{#if version.specification_url}
							<LinkExternal
								href={version.specification_url}
								text="{version.name} {m.Spec()}"
								icon={GitBranch}
								title={m.View_specification({ name: version.name })}
							/>
						{/if}
					</div>
				{/if}
			{/snippet}

			<StandardAndVersionField {form} name="standard_and_version" />
		</PageCardSection>

		<PageCardSection title={m.Check_Metadata()} description={m.Check_metadata_description()}>
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
				<Field {form} name="name" options={{ label: m.Name() }} />

				<LogoField {form} name="logo" label="AO" initialPreviewUrl={originalLogoUrl} />

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

			<CodeEditorField
				{form}
				name="input_json_sample"
				options={{
					lang: 'json',
					minHeight: 200,
					label: 'Configuration example',
					description: 'Paste here a example JSON of the content of the config'
				}}
			/>
		</PageCardSection>

		<PageCardSection title={m.Make_public()} description={m.Make_public_section_description()}>
			<CheckboxField
				{form}
				name="published"
				options={{
					label: `${m.Make_public()}`
				}}
			/>
		</PageCardSection>

		{#snippet submitButtonContent()}
			<PlusIcon />
			{currentLabels.submitButton}
		{/snippet}
	</Form>
</FocusPageLayout>

{#snippet yamlFieldLabelRight()}
	<Button variant="secondary" onclick={startYamlUpload}>
		<UploadIcon />
		{m.Upload_yaml()}
	</Button>
{/snippet}
