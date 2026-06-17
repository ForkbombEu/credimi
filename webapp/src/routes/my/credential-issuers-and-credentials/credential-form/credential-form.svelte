<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import { yaml } from '@codemirror/lang-yaml';
	import CollectionLogoField from '$lib/components/collection-logo-field.svelte';
	import QrFieldWrapper from '$lib/layout/qr-field-wrapper.svelte';
	import { optionalSecretsYamlSchema, refineAsStepciYaml } from '$lib/utils';
	import { z } from 'zod/v3';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type {
		CredentialIssuersResponse,
		CredentialsFormData,
		CredentialsResponse
	} from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import SubmitButton from '@/collections-components/manager/record-actions/submit-button.svelte';
	import { FormError } from '@/forms';
	import { CodeEditorField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	import QrGenerationField, { type FieldMode } from './qr-generation-field.svelte';

	//

	type Props = {
		credential?: CredentialsResponse;
		credentialIssuer: CredentialIssuersResponse;
		onSuccess?: () => void;
	};

	let { credential, credentialIssuer, onSuccess }: Props = $props();

	const mode = $derived(credential ? 'edit' : 'create');

	type Field = keyof CredentialsFormData;
	const exclude: Field[] = $derived.by(() => {
		const commonFields: Field[] = [
			'json',
			'owner',
			'conformant',
			'imported',
			'published',
			'canonified_name',
			'logo_url'
		];
		const editFields: Field[] = ['format', 'display_name', 'locale', 'logo_url', 'name'];
		if (mode === 'edit' && credential?.imported) {
			commonFields.push(...editFields);
		}
		return commonFields;
	});

	//

	let activeTab = $state<FieldMode>('static');
</script>

<CollectionForm
	collection="credentials"
	recordId={credential?.id}
	initialData={credential}
	schemaContext={{
		parentId: credentialIssuer.id,
		excludeId: credential?.id
	}}
	uiOptions={{
		hide: ['submit_button', 'error']
	}}
	refineSchema={(schema) =>
		schema.extend({
			yaml: refineAsStepciYaml(z.string().optional()) as unknown as z.ZodOptional<z.ZodString>,
			secrets: optionalSecretsYamlSchema as unknown as z.ZodOptional<z.ZodString>
		})}
	fieldsOptions={{
		exclude,
		order: ['name', 'display_name', 'description', 'deeplink', 'logo'],
		labels: {
			published: m.Publish_to_hub(),
			deeplink: m.QR_Code_Generation()
		},
		snippets: {
			description,
			deeplink: qr_generation,
			logo
		},
		hide: {
			yaml: credential?.yaml,
			secrets: '',
			credential_issuer: credentialIssuer.id,
			display_name: credential?.display_name
		},
		placeholders: {
			name: 'e.g. Above 18',
			format: 'e.g. jwt_vc_json',
			locale: 'e.g. en-US'
		}
	}}
	beforeSubmit={(data) => {
		if (activeTab === 'static') {
			data.yaml = '';
		}
		return data;
	}}
	{onSuccess}
>
	<FormError />
	<SubmitButton>
		{m.Save()}
	</SubmitButton>
</CollectionForm>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet qr_generation({ form }: FieldSnippetOptions<'credentials'>)}
	{#snippet secrets_editor()}
		<CodeEditorField
			{form}
			name="secrets"
			options={{
				lang: yaml(),
				minHeight: 160,
				maxHeight: 300,
				label: m.Secrets(),
				description: m.Secrets_field_description()
			}}
		/>
	{/snippet}

	<QrFieldWrapper label={m.Credential_Deeplink()}>
		<QrGenerationField
			form={form as unknown as SuperForm<{ deeplink: string; yaml: string; secrets?: string }>}
			{credential}
			{credentialIssuer}
			bind:activeTab
			secretsEditor={secrets_editor}
		/>
	</QrFieldWrapper>
{/snippet}

{#snippet logo()}
	<CollectionLogoField />
{/snippet}
