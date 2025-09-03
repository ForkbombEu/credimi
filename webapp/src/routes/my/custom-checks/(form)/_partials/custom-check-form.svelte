<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StandardsWithTestSuites } from '$lib/standards';

	import { yaml } from '@codemirror/lang-yaml';
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import PageCardSection from '$lib/layout/page-card-section.svelte';
	import StandardAndVersionField from '$lib/standards/standard-and-version-field.svelte';
	import { jsonStringSchema, yamlStringSchema } from '$lib/utils';
	import { String } from 'effect';
	import { run } from 'json_typegen_wasm';
	import _ from 'lodash';
	import { ExternalLink, GitBranch, HelpCircle, Home, PlusIcon, UploadIcon } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { fromStore } from 'svelte/store';
	import { zod } from 'sveltekit-superforms/adapters';

	import type { CustomChecksRecord, CustomChecksResponse } from '@/pocketbase/types';

	import { removeEmptyValues } from '@/collections-components/form';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { createForm, Form } from '@/forms';
	import {
		CheckboxField,
		CodeEditorField,
		Field,
		FileField,
		TextareaField
	} from '@/forms/fields';
	import { goto, m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import { getExceptionMessage } from '@/utils/errors.js';
	import { readFileAsDataURL, readFileAsString } from '@/utils/files.js';

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
			yaml: yamlStringSchema,
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
		let input_json_sample = JSON.stringify(record.input_json_sample, null, 2);
		if (input_json_sample == 'null') input_json_sample = '';
		return {
			..._.omit(record, 'logo', 'input_json_schema'),
			input_json_sample
		};
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

	let avatarPreviewUrl = $derived(originalLogoUrl);

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

	$effect(() => {
		const logo = formState.current.logo;
		if (logo) {
			readFileAsDataURL(logo).then((dataURL) => {
				avatarPreviewUrl = dataURL;
			});
		} else {
			avatarPreviewUrl = originalLogoUrl;
		}
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
							<a
								href={standard.standard_url}
								class="inline-flex items-center gap-1 rounded-md bg-[hsl(var(--link-help-background))] px-2 py-1 text-xs text-[hsl(var(--link-help-foreground))] transition-colors hover:bg-[hsl(var(--link-help-background))]/80"
								target="_blank"
								rel="noopener noreferrer"
								title={m.Learn_about_standard({ name: standard.name })}
							>
								<HelpCircle class="h-3 w-3" />
								{standard.name}
								{m.Standard()}
								<ExternalLink class="h-3 w-3" />
							</a>
						{/if}

						{#if version.specification_url}
							<a
								href={version.specification_url}
								class="inline-flex items-center gap-1 rounded-md bg-[hsl(var(--link-homepage-background))] px-2 py-1 text-xs text-[hsl(var(--link-homepage-foreground))] transition-colors hover:bg-[hsl(var(--link-homepage-background))]/80"
								target="_blank"
								rel="noopener noreferrer"
								title={m.View_specification({ name: version.name })}
							>
								<GitBranch class="h-3 w-3" />
								{version.name}
								{m.Spec()}
								<ExternalLink class="h-3 w-3" />
							</a>
						{/if}

						<a
							href="/marketplace?type=custom_checks&standard={standard.uid}"
							class="inline-flex items-center gap-1 rounded-md bg-[hsl(var(--link-subtle-background))] px-2 py-1 text-xs text-[hsl(var(--link-subtle-foreground))] transition-colors hover:bg-[hsl(var(--link-subtle-background))]/80"
							title={m.Browse_custom_checks_for_standard({ name: standard.name })}
						>
							<Home class="h-3 w-3" />
							{m.Browse_All()}
							<ExternalLink class="h-3 w-3" />
						</a>
					</div>
				{/if}
			{/snippet}

			<StandardAndVersionField {form} name="standard_and_version" />
		</PageCardSection>

		<PageCardSection title={m.Check_Metadata()} description={m.Check_metadata_description()}>
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
				<Field {form} name="name" options={{ label: m.Name() }} />

				<div class="flex items-start gap-4">
					<div class="grow">
						<FileField
							{form}
							variant="outline"
							name="logo"
							options={{ label: m.Upload_logo() }}
						>
							<UploadIcon />
							<T>{m.Upload_logo()}</T>
						</FileField>
					</div>
					<div class="pt-2">
						<Avatar
							src={avatarPreviewUrl}
							alt={record?.name}
							class="size-16 rounded-md border"
						/>
					</div>
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
				name="public"
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
