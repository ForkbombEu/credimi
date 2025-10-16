<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import { stepciYamlSchema } from '$lib/utils';
	import { ZodOptional, ZodString } from 'zod';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type {
		CredentialIssuersResponse,
		CredentialsFormData,
		CredentialsResponse
	} from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
	import { FormError, SubmitButton } from '@/forms';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	import LogoField from '../_wallets/logo-field.svelte';
	import QrGenerationField, { type FieldMode } from './deeplink-tabs/index.svelte';

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
			'canonified_name'
		];
		// For imported credentials, exclude optional fields (NOT name - it's required!)
		const editFields: Field[] = ['format', 'display_name', 'locale'];
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
			yaml: stepciYamlSchema.optional() as unknown as ZodOptional<ZodString>
		})}
	fieldsOptions={{
		exclude,
		order: ['name', 'description', 'deeplink'],
		labels: {
			published: m.Publish_to_marketplace(),
			deeplink: m.QR_Code_Generation()
		},
		snippets: {
			description,
			deeplink: qr_generation,
			logo: logo_snippet
		},
		hide: {
			yaml: credential?.yaml,
			credential_issuer: credentialIssuer.id,
			display_name: credential?.display_name,
			// Hide logo_url field - it's rendered inside LogoField snippet instead
			logo_url: credential?.logo_url,
			// For imported credentials, hide name field but keep it in form data (it's required)
			...(mode === 'edit' && credential?.imported ? { name: credential.name } : {})
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
	<MarkdownField {form} name="description" height={200} />
{/snippet}

{#snippet logo_snippet({ form }: FieldSnippetOptions<'credentials'>)}
	<LogoField {form} recordResponse={credential} />
{/snippet}

{#snippet qr_generation({ form }: FieldSnippetOptions<'credentials'>)}
	<div>
		<T tag="h3" class="mb-6">{m.Credential_Deeplink()}</T>

		<QrGenerationField
			form={form as unknown as SuperForm<{ deeplink: string; yaml: string }>}
			{credential}
			{credentialIssuer}
			bind:activeTab
		/>
	</div>

	<div class="py-2">
		<Separator />
	</div>

	<T tag="h3">{m.Metadata()}</T>
{/snippet}
