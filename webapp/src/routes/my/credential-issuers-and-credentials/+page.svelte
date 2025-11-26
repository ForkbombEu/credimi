<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CollectionLogoField from '$lib/components/collection-logo-field.svelte';

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
		exclude: [
			'owner',
			'url',
			'published',
			'imported',
			'logo_url',
			'canonified_name',
			'workflow_url'
		],
		snippets: {
			logo
		}
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

{#snippet logo()}
	<CollectionLogoField />
{/snippet}
