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
	import { Separator } from '@/components/ui/separator';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	import QrGenerationField from './qr-generation-field/index.svelte';

	//

	type Props = {
		credential: CredentialsResponse;
		credentialIssuer: CredentialIssuersResponse;
		onSuccess?: () => void;
		mode?: 'edit' | 'create';
	};

	let { credential, credentialIssuer, onSuccess, mode = 'edit' }: Props = $props();

	type Field = keyof CredentialsFormData;
	const exclude: Field[] = $derived.by(() => {
		const commonFields: Field[] = [
			'format',
			'issuer_name',
			'type',
			'name',
			'locale',
			'logo',
			'credential_issuer',
			'json',
			'key',
			'owner',
			'conformant',
			'published',
			'imported',
			'yaml'
		];
		if (mode === 'create') {
			commonFields.push('yaml');
		}
		return commonFields;
	});
</script>

<CollectionForm
	collection="credentials"
	recordId={credential.id}
	initialData={credential}
	fieldsOptions={{
		exclude,
		order: ['deeplink'],
		labels: {
			published: m.Publish_to_marketplace(),
			deeplink: 'QR Code Generation'
		},
		snippets: {
			description,
			deeplink: qr_generation
		}
	}}
	{onSuccess}
/>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet qr_generation({ form }: FieldSnippetOptions<'credentials'>)}
	<QrGenerationField {form} deeplinkName="deeplink" yaml="yaml" {credential} {credentialIssuer} />

	<div class="py-2">
		<Separator />
	</div>
{/snippet}
