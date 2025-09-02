<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { CredentialsResponse, CredentialIssuersResponse } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components/index.js';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	import DeeplinkField from './deeplink-field.svelte';

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
			'published'
		],
		order: ['deeplink'],
		labels: {
			published: m.Publish_to_marketplace()
		},
		snippets: {
			description,
			deeplink
		}
	}}
	onSuccess={() => {
		onSuccess?.();
	}}
	uiOptions={{
		toastText: m.Credential_updated_successfully()
	}}
/>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet deeplink({ form }: FieldSnippetOptions<'credentials'>)}
	<DeeplinkField {form} {credential} {credentialIssuer} name="deeplink" />
{/snippet}
