<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';

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
	};

	let { credential, credentialIssuer, onSuccess }: Props = $props();
</script>

<CollectionForm
	collection="credentials"
	recordId={credential.id}
	initialData={credential}
	fieldsOptions={{
		exclude: [
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
		],
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
