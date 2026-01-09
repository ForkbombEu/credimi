<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import CollectionLogoField from '$lib/components/collection-logo-field.svelte';
	import QrFieldWrapper from '$lib/layout/qr-field-wrapper.svelte';
	import { refineAsStepciYaml } from '$lib/utils';
	import { z } from 'zod';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type {
		CredentialIssuersResponse,
		CredentialsFormData,
		CredentialsResponse
	} from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import { FormError, SubmitButton } from '@/forms';
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
			yaml: refineAsStepciYaml(z.string().optional()) as unknown as z.ZodOptional<z.ZodString>
		})}
	fieldsOptions={{
		exclude,
		order: ['name', 'display_name', 'description', 'deeplink', 'logo'],
		labels: {
			published: m.Publish_to_marketplace(),
			deeplink: m.QR_Code_Generation()
		},
		snippets: {
			description,
			deeplink: qr_generation,
			logo
		},
		hide: {
			yaml: credential?.yaml,
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
	<div
		class="sticky bottom-0 -mx-6 -mt-6 flex justify-end border-t bg-white/70 px-6 py-2 backdrop-blur-sm"
	>
		<SubmitButton>
			{m.Save()}
		</SubmitButton>
	</div>
</CollectionForm>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet qr_generation({ form }: FieldSnippetOptions<'credentials'>)}
	<QrFieldWrapper label={m.Credential_Deeplink()}>
		<QrGenerationField
			form={form as unknown as SuperForm<{ deeplink: string; yaml: string }>}
			{credential}
			{credentialIssuer}
			bind:activeTab
		/>
	</QrFieldWrapper>
{/snippet}

{#snippet logo()}
	<CollectionLogoField />
{/snippet}
