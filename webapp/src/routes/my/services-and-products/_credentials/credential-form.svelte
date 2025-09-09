<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
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

	import QrGenerationField from './qr-generation-field/index.svelte';

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
			'credential_issuer',
			'json',
			'owner',
			'conformant',
			'imported',
			'published'
		];
		const editFields: Field[] = [
			'format',
			'issuer_name',
			'type',
			'name',
			'locale',
			'logo',
			'key'
		];
		if (mode === 'edit') {
			commonFields.push(...editFields);
		}
		// else if (mode === 'create') {
		// 	commonFields.push('yaml');
		// }
		return commonFields;
	});
</script>

<CollectionForm
	collection="credentials"
	recordId={credential?.id}
	initialData={credential}
	uiOptions={{
		hide: ['submit_button', 'error']
	}}
	fieldsOptions={{
		exclude,
		order: ['deeplink', 'name', 'description'],
		labels: {
			published: m.Publish_to_marketplace(),
			deeplink: 'QR Code Generation'
		},
		snippets: {
			description,
			deeplink: qr_generation
		},
		hide: {
			yaml: credential?.yaml
		}
	}}
	{onSuccess}
>
	<FormError />
	<div
		class="sticky bottom-0 -mx-6 -mt-6 flex justify-end border-t bg-white/70 px-6 py-2 backdrop-blur-sm"
	>
		<SubmitButton>
			{#if mode === 'edit'}
				{m.Edit_credential()}
			{:else if mode === 'create'}
				{m.Create_credential()}
			{/if}
		</SubmitButton>
	</div>
</CollectionForm>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet qr_generation({ form }: FieldSnippetOptions<'credentials'>)}
	<div>
		<T tag="h3" class="mb-6">Credential Deeplink</T>

		<QrGenerationField
			{form}
			deeplinkName="deeplink"
			yaml="yaml"
			{credential}
			{credentialIssuer}
		/>
	</div>

	<div class="py-2">
		<Separator />
	</div>

	<T tag="h3">Metadata</T>
{/snippet}
