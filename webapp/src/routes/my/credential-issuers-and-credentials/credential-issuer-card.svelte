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

	import type {
		CredentialIssuersResponse,
		CredentialsResponse,
		OrganizationsResponse
	} from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import CredentialForm from './credential-form/credential-form.svelte';

	//

	type Props = {
		credentialIssuer: CredentialIssuersResponse;
		organization: OrganizationsResponse;
	};

	let { credentialIssuer: issuer, organization }: Props = $props();

	//

	async function refreshCredentialIssuer(url: string) {
		runWithLoading({
			fn: async () => {
				await pb.send('/api/credentials_issuers/start-check', {
					method: 'POST',
					body: {
						credentialIssuerUrl: url
					}
				});
			},
			loadingText: 'Updating credential issuer...',
			errorText: 'Failed to refresh credential issuer'
		});
	}

	function getCredentialPublicUrl(credential: CredentialsResponse) {
		return `/marketplace/credentials/${organization.canonified_name}/${issuer.canonified_name}/${credential.canonified_name}`;
	}
</script>

<DashboardCard
	record={issuer}
	avatar={(i) => (i.logo ? pb.files.getURL(i, i.logo) : i.logo_url)}
	badge={issuer.imported ? m.Imported() : undefined}
	links={{
		URL: issuer.url,
		[m.Repository()]: issuer.repo_url,
		[m.Homepage()]: issuer.homepage_url,
		[m.Import_results()]: issuer.workflow_url
	}}
	path={[organization.canonified_name, issuer.canonified_name]}
>
	{#snippet actions()}
		<Button
			variant="outline"
			size="icon"
			disabled={!issuer.imported}
			onclick={() => refreshCredentialIssuer(issuer.url)}
		>
			<RefreshCwIcon />
		</Button>
	{/snippet}

	{#snippet content()}
		{@render credentialsManager(issuer)}
	{/snippet}
</DashboardCard>

{#snippet credentialsManager(issuer: CredentialIssuersResponse)}
	<CollectionManager
		collection="credentials"
		queryOptions={{ filter: `credential_issuer = '${issuer.id}'`, perPage: 10 }}
		hide={['empty_state', 'pagination']}
	>
		{#snippet createForm({ closeSheet })}
			<CredentialForm credentialIssuer={issuer} onSuccess={closeSheet} />
		{/snippet}

		{#snippet editForm({ record: credential, closeSheet })}
			<CredentialForm {credential} credentialIssuer={issuer} onSuccess={closeSheet} />
		{/snippet}

		{#snippet top()}
			<DashboardCardManagerTop
				label={m.Credentials()}
				buttonText={m.Add_new_credential()}
				recordCreateOptions={{
					formTitle: `${m.Credential_issuer()}: ${issuer.name} — ${m.Add_new_credential()}`
				}}
			/>
		{/snippet}

		{#snippet records({ records: credentials, Pagination })}
			<DashboardCardManagerUI
				records={credentials}
				nameField="display_name"
				fallbackNameField="name"
				hideClone={issuer.imported}
				path={(r) => [
					organization.canonified_name,
					issuer.canonified_name,
					r.canonified_name
				]}
				publicUrl={getCredentialPublicUrl}
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
