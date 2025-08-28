<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionForm } from '@/collections-components';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import ImportCredentialIssuer from './import-credential-issuer.svelte';

	//

	type Props = {
		organizationId: string;
	};

	let { organizationId }: Props = $props();
</script>

<Sheet>
	ciao

	{#snippet content({ closeSheet })}
		<ImportCredentialIssuer {organizationId}>
			{#snippet after({ formState: { loading, credentialIssuer } })}
				{#if !credentialIssuer}
					<CollectionForm
						collection="credential_issuers"
						fieldsOptions={{
							hide: {
								owner: organizationId
							},
							exclude: ['published']
						}}
						onSuccess={closeSheet}
						uiOptions={{
							showToastOnSuccess: true
						}}
					></CollectionForm>
				{:else if loading}
					<div>Loading...</div>
				{:else}
					<CollectionForm
						collection="credential_issuers"
						recordId={credentialIssuer.id}
						initialData={credentialIssuer}
						fieldsOptions={{
							hide: {
								owner: organizationId
							},
							exclude: ['owner', 'url']
						}}
						uiOptions={{
							showToastOnSuccess: true
						}}
					></CollectionForm>
				{/if}
			{/snippet}
		</ImportCredentialIssuer>
	{/snippet}
</Sheet>
