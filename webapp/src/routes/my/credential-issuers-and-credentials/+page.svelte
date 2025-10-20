<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import LabelLink from '$lib/layout/label-link.svelte';
	import PublishedSwitch from '$lib/layout/published-switch.svelte';
	import { runWithLoading } from '$lib/utils';
	import { RefreshCwIcon } from 'lucide-svelte';

	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import { RecordClone, RecordCreate, RecordDelete } from '@/collections-components/manager';
	import Button from '@/components/ui-custom/button.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import CredentialForm from './credential-form.svelte';
	import CredentialIssuerForm from './credential-issuer-form/credential-issuer-form.svelte';
	import EditCredentialDialog from './edit-credential-dialog.svelte';
	import { fetchCredentialIssuer } from './utils';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	//

	function getCredentialCopyText(
		credential: CredentialsResponse,
		credentialIssuer: CredentialIssuersResponse
	) {
		const organizationName =
			organization?.canonified_name ||
			organization?.name ||
			organization.id ||
			'Unknown Organization';
		const credentialIssuerName =
			credentialIssuer.canonified_name ||
			credentialIssuer.name ||
			'Unknown Credential Issuer';
		const credentialName =
			credential.canonified_name ||
			credential.display_name ||
			credential.name ||
			'Unknown Credential';

		return `${organizationName}/${credentialIssuerName}/${credentialName}`;
	}

	async function refreshCredentialIssuer(url: string) {
		runWithLoading({
			fn: () => fetchCredentialIssuer(url),
			loadingText: 'Updating credential issuer...',
			errorText: 'Failed to refresh credential issuer'
		});
	}

	const copyTooltipText = `${m.Copy()} ${m.Organization()}/${m.Credential_issuer()}/${m.Credential()}`;

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
		{#snippet top({ records, totalRecords, pageRange })}
			<div class="flex items-center justify-between">
				{#if records.length > 0}
					<T class="text-sm font-semibold">
						{m.count_available_credentials({
							number: `${pageRange} / ${totalRecords}`
						})}
					</T>
				{:else}
					<T class="text-gray-300">{m.No_credentials_available()}</T>
				{/if}
				<RecordCreate>
					{#snippet button({ triggerAttributes, icon })}
						<BlueButton {icon} text={m.Add_new_credential()} {...triggerAttributes} />
					{/snippet}

					{#snippet form({ closeSheet })}
						<CredentialForm credentialIssuer={record} onSuccess={closeSheet} />
					{/snippet}
				</RecordCreate>
			</div>
		{/snippet}

		{#snippet records({ records: credentials, reloadRecords, Pagination })}
			<ul class="space-y-2">
				{#each credentials as credential (credential.id)}
					<li
						class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2 hover:ring-2"
					>
						<LabelLink
							label={credential.display_name || credential.name}
							href="/marketplace/credentials/{organization?.canonified_name}/{credential.canonified_name}"
							published={credential.published && record.published}
							class="min-w-0 flex-1 break-words"
						/>

						<div class="flex items-center gap-2">
							{#if credential.imported}
								<Badge variant="secondary">{m.Imported()}</Badge>
							{/if}

							<Tooltip>
								<CopyButtonSmall
									textToCopy={getCredentialCopyText(credential, record)}
									square
								/>
								{#snippet content()}
									<p>{copyTooltipText}</p>
								{/snippet}
							</Tooltip>

							<PublishedSwitch
								size="sm"
								disabled={!record.published}
								record={credential}
								field="published"
							/>

							{#if !credential.imported}
								<RecordClone
									collectionName="credentials"
									record={credential}
									onSuccess={reloadRecords}
								/>
							{/if}

							<EditCredentialDialog {credential} credentialIssuer={record} />

							<RecordDelete record={credential}>
								{#snippet button({ triggerAttributes, icon })}
									<IconButton
										variant="outline"
										size="sm"
										{icon}
										{...triggerAttributes}
									/>
								{/snippet}
							</RecordDelete>
						</div>
					</li>
				{/each}
			</ul>
			<Pagination scrollToTop={false} />
		{/snippet}
	</CollectionManager>
{/snippet}
