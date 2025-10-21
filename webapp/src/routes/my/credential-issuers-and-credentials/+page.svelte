<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import IssuerCard from './credential-issuer-card.svelte';
	import CredentialIssuerForm from './credential-issuer-form/credential-issuer-form.svelte';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Credential Issuers and Credentials', right: navbarRight });
</script>

<CollectionManager
	collection="credential_issuers"
	queryOptions={{
		filter: `owner.id = '${organization.id}'`,
		sort: ['created', 'DESC']
	}}
	editFormFieldsOptions={{
		exclude: ['owner', 'url', 'published', 'imported', 'canonified_name', 'workflow_url']
	}}
>
	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as credentialIssuer (credentialIssuer.id)}
				<IssuerCard {credentialIssuer} {organization} />
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet navbarRight()}
	<CredentialIssuerForm organizationId={organization.id} />
{/snippet}
