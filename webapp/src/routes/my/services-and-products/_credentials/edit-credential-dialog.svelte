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
	import { CodeEditorField } from '@/forms/fields';
	import { yaml as yamlLang } from '@codemirror/lang-yaml';
	import Button from '@/components/ui-custom/button.svelte';
	import { QrCode } from '@/qr';
	import { processYamlAndExtractCredentialOffer } from '$lib/compliance';

	type Props = {
		credential: CredentialsRecord;
		credentialIssuer: CredentialIssuersResponse;
		onSuccess: () => void;
	};

	let { credential, credentialIssuer, onSuccess }: Props = $props();

	// State for compliance testing
	let isSubmittingCompliance = $state(false);
	let credentialOffer = $state<string | null>(null);

	async function startComplianceTest(yamlContent: string) {
		if (!yamlContent.trim()) {
			toast.error('YAML configuration is required');
			return;
		}

		isSubmittingCompliance = true;

		try {
			const result = await processYamlAndExtractCredentialOffer(yamlContent);
			
			if (result.success) {
				if (result.credentialOffer) {
					credentialOffer = result.credentialOffer;
					toast.success('Compliance test completed successfully with credential offer!');
				} else {
					toast.success('Compliance test completed successfully!');
				}
			} else {
				toast.error(result.error || 'Compliance test failed');
			}
		} catch (error) {
			console.error('Failed to start compliance test:', error);
			toast.error('Failed to start compliance test');
		} finally {
			isSubmittingCompliance = false;
		}
	}
</script>

<Sheet title="{m.Edit_credential()}: {credential.name || credential.key}">
	{#snippet trigger({ sheetTriggerAttributes, openSheet })}
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
				order: ['deeplink', 'yaml'],
				labels: {
					published: m.Publish_to_marketplace(),
					yaml: m.YAML_Configuration()
				},
				snippets: {
					description,
					deeplink,
					yaml
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

{#snippet yaml({ form, formData }: FieldSnippetOptions<'credentials'>)}
	<div class="space-y-3">
		<CodeEditorField
			{form}
			name="yaml"
			options={{
				lang: yamlLang(),
				minHeight: 200,
				label: m.YAML_Configuration(),
				description: 'Provide the credential configuration in YAML format'
			}}
		/>
		<Button
			type="button"
			variant="secondary"
			disabled={!formData.yaml || isSubmittingCompliance}
			onclick={() => startComplianceTest(formData.yaml as string)}
			class="w-full"
		>
			{isSubmittingCompliance ? 'Starting Compliance Test...' : 'Start Compliance Test'}
		</Button>
		
		{#if credentialOffer}
			<div class="border rounded-lg p-4 space-y-3">
				<h4 class="font-medium text-sm">Credential Offer</h4>
				<div class="flex flex-col md:flex-row items-stretch gap-4">
					<QrCode src={credentialOffer} cellSize={10} class="size-60 rounded-md border" />
					<div class="max-w-60 break-all text-xs">
						<a href={credentialOffer} target="_blank" rel="noopener">{credentialOffer}</a>
					</div>
				</div>
			</div>
		{/if}
	</div>
{/snippet}
