<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import ImportCredentialIssuer from './import-credential-issuer.svelte';
	import type { CredentialIssuersResponse } from '@/pocketbase/types';
	import { CollectionForm } from '@/collections-components';

	//

	type Props = {
		organizationId: string;
	};

	let { organizationId }: Props = $props();

	let credentialIssuerPromise = $state<Promise<CredentialIssuersResponse>>();
</script>

<Sheet>
	ciao

	{#snippet content()}
		<ImportCredentialIssuer {organizationId}>
			{#snippet after({ formState: { loading, credentialIssuer } })}
				{#if loading}
					<div>Loading...</div>
				{:else if !credentialIssuer}
					<CollectionForm
						collection="credential_issuers"
						fieldsOptions={{
							hide: {
								owner: organizationId
							},
							exclude: ['published']
						}}
					></CollectionForm>
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
					></CollectionForm>
				{/if}
			{/snippet}
		</ImportCredentialIssuer>
	{/snippet}
</Sheet>
