<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil } from 'lucide-svelte';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
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

<Sheet title="{m.Edit_credential()}: {credential.name || credential.key}">
	{#snippet trigger({ sheetTriggerAttributes })}
		<IconButton size="sm" variant="outline" icon={Pencil} {...sheetTriggerAttributes} />
	{/snippet}

	{#snippet content({ closeSheet })}
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
					published: m.Publish_to_marketplace(),
					deeplink: 'QR Code Generation'
				},
				snippets: {
					description,
					deeplink: qr_generation
				}
			}}
			onSuccess={() => {
				closeSheet();
				onSuccess?.();
			}}
		/>
	{/snippet}
</Sheet>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet qr_generation({ form }: FieldSnippetOptions<'credentials'>)}
	<QrGenerationField {form} deeplinkName="deeplink" yaml="yaml" {credential} {credentialIssuer} />
{/snippet}
