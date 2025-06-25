<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionForm } from '@/collections-components';
	import type { CredentialIssuersResponse, CredentialsRecord } from '@/pocketbase/types';
	import { Pencil } from 'lucide-svelte';
	import { m } from '@/i18n';
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import DeeplinkField from './deeplink-field.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { toast } from 'svelte-sonner';

	type Props = {
		credential: CredentialsRecord;
		credentialIssuer: CredentialIssuersResponse;
		onSuccess: () => void;
	};

	let { credential, credentialIssuer, onSuccess }: Props = $props();
</script>

<Sheet title="{m.Edit_credential()}: {credential.name}">
	{#snippet trigger({ sheetTriggerAttributes, openSheet })}
		<IconButton variant="outline" icon={Pencil} {...sheetTriggerAttributes} />
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
					'conformant'
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
				toast.success(m.Credential_updated_successfully());
				closeSheet();
				onSuccess();
			}}
		/>
	{/snippet}
</Sheet>

{#snippet description({ form }: FieldSnippetOptions<'credentials'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet deeplink({ form }: FieldSnippetOptions<'credentials'>)}
	<DeeplinkField {form} {credential} {credentialIssuer} name="deeplink" />
{/snippet}
