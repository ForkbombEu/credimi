<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCardManagerTop from '$lib/layout/dashboard-card-manager-top.svelte';
	import DashboardCardManagerUI from '$lib/layout/dashboard-card-manager-ui.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { runWithLoading } from '$lib/utils';
	import { RefreshCwIcon } from 'lucide-svelte';

	import type { CredentialIssuersResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import CredentialForm from './credential-form.svelte';
	import CredentialIssuerForm from './credential-issuer-form/credential-issuer-form.svelte';
	import { fetchCredentialIssuer } from './utils';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	//

	async function refreshCredentialIssuer(url: string) {
		runWithLoading({
			fn: () => fetchCredentialIssuer(url),
			loadingText: 'Updating credential issuer...',
			errorText: 'Failed to refresh credential issuer'
		});
	}

	setDashboardNavbar({ title: 'Credential Issuers and Credentials', right: navbarRight });
</script>

<CollectionManager
	collection="credential_issuers"
	queryOptions={{
		filter: `owner.id = '${organization.id}'`,
		sort: ['created', 'DESC']
	}}
	editFormFieldsOptions={{
		exclude: ['owner', 'url', 'published', 'imported', 'canonified_name']
	}}
>
	{#snippet records({ records })}
		<div class="flex flex-col items-stretch gap-6">
			{#each records as record (record)}
				<DashboardCard
					{record}
					{organization}
					avatar={(r) => r.logo_url}
					badge={record.imported ? m.Imported() : undefined}
					links={{
						URL: record.url,
						[m.Repository()]: record.repo_url,
						[m.Homepage()]: record.homepage_url,
						[m.Import_results()]: record.workflow_url
					}}
				>
					{#snippet actions()}
						<Button
							variant="outline"
							size="icon"
							disabled={!record.imported}
							onclick={() => refreshCredentialIssuer(record.url)}
						>
							<RefreshCwIcon />
						</Button>
					{/snippet}

					{#snippet content()}
						{@render CredentialsManager(record)}
					{/snippet}
				</DashboardCard>
			{/each}
		</div>
	{/snippet}
</CollectionManager>

<!--  -->

{#snippet navbarRight()}
	<CredentialIssuerForm organizationId={organization.id} />
{/snippet}

{#snippet CredentialsManager(record: CredentialIssuersResponse)}
	<CollectionManager
		collection="credentials"
		queryOptions={{ filter: `credential_issuer = '${record.id}'`, perPage: 10 }}
		hide={['empty_state', 'pagination']}
	>
		{#snippet createForm({ closeSheet })}
			<CredentialForm credentialIssuer={record} onSuccess={closeSheet} />
		{/snippet}

		{#snippet editForm({ record: credential, closeSheet })}
			<CredentialForm {credential} credentialIssuer={record} onSuccess={closeSheet} />
		{/snippet}

		{#snippet top()}
			<DashboardCardManagerTop
				label={m.Credentials()}
				buttonText={m.Add_new_credential()}
				recordCreateOptions={{
					formTitle: `${m.Credential_issuer()}: ${record.name} — ${m.Add_new_credential()}`
				}}
			/>
		{/snippet}

		{#snippet records({ records: credentials, Pagination })}
			<DashboardCardManagerUI
				records={credentials}
				nameField="display_name"
				hideClone={record.imported}
				publicUrl={(c) =>
					`/marketplace/credentials/${organization.canonified_name}/${c.canonified_name}`}
				textToCopy={(c) =>
					`${organization.canonified_name}/${record.canonified_name}/${c.canonified_name}`}
			>
				{#snippet actions({ record: credential })}
					{#if credential.imported}
						<Badge variant="secondary">{m.Imported()}</Badge>
					{/if}
				{/snippet}
			</DashboardCardManagerUI>

			<Pagination scrollToTop={false} />
		{/snippet}
	</CollectionManager>
{/snippet}
