<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { runWithLoading } from '$lib/utils';
	import { String } from 'effect';
	import { Eye, EyeOff, RefreshCwIcon } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';
	import type {
		CredentialIssuersResponse,
		CredentialsResponse,
		OrganizationsResponse
	} from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import {
		RecordClone,
		RecordCreate,
		RecordDelete,
		RecordEdit
	} from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Card } from '@/components/ui/card';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import LabelLink from '../_partials/label-link.svelte';
	import CredentialForm from './credential-form.svelte';
	import CredentialIssuerForm from './credential-issuer-form/credential-issuer-form.svelte';
	import EditCredentialDialog from './edit-credential-dialog.svelte';
	import { fetchCredentialIssuer } from './utils';

	//

	type Props = {
		organizationId: string;
		organization?: OrganizationsResponse;
		id?: string;
	};

	let { organizationId, organization, id }: Props = $props();

	//

	function getCredentialCopyText(
		credential: CredentialsResponse,
		credentialIssuer: CredentialIssuersResponse
	) {
		const organizationName =
			organization?.canonified_name ||
			organization?.name ||
			organizationId ||
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

	async function updateCredentialIssuerPublished(
		credentialIssuer: CredentialIssuersResponse,
		published: boolean
	) {
		const res = await pb
			.collection('credential_issuers')
			.update(credentialIssuer.id, { published });
		credentialIssuer.published = res.published;
	}

	async function updateCredentialPublished(credential: CredentialsResponse, published: boolean) {
		const res = await pb.collection('credentials').update(credential.id, { published });
		credential.published = res.published;
	}

	async function refreshCredentialIssuer(url: string) {
		runWithLoading({
			fn: () => fetchCredentialIssuer(url),
			loadingText: 'Updating credential issuer...',
			errorText: 'Failed to refresh credential issuer'
		});
	}

	const copyTooltipText = `${m.Copy()} ${m.Organization()}/${m.Credential_issuer()}/${m.Credential()}`;
</script>

<CollectionManager
	collection="credential_issuers"
	queryOptions={{
		filter: `owner.id = '${organizationId}'`,
		sort: ['created', 'DESC']
	}}
	editFormFieldsOptions={{
		exclude: ['owner', 'url', 'published', 'imported', 'canonified_name']
	}}
>
	{#snippet top({ Header })}
		<Header title={m.Credential_issuers()} {id}>
			{#snippet right()}
				<CredentialIssuerForm {organizationId} />
			{/snippet}
		</Header>
	{/snippet}

	{#snippet records({ records, reloadRecords })}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each records as record (record)}
				{@render CredentialIssuerCard(record, reloadRecords)}
			{/each}
		</div>
	{/snippet}
</CollectionManager>

<!--  -->

{#snippet CredentialIssuerCard(record: CredentialIssuersResponse, onEditSuccess: () => void)}
	<Card class="!p-4">
		<div class="space-y-4">
			<div class="flex items-start justify-between gap-6">
				<Avatar
					src={record.logo_url}
					class="rounded-sm border"
					fallback={record.name.slice(0, 2)}
				/>

				<div class="flex w-0 grow items-center gap-2">
					<T class="truncate font-bold">
						<LabelLink
							label={String.isNonEmpty(record.name) ? record.name : '[no_title]'}
							href="/marketplace/credential_issuers/{organization?.canonified_name}/{record.canonified_name}"
							published={record.published}
						/>
					</T>
					{#if record.imported}
						<Badge variant="secondary">{m.Imported()}</Badge>
					{/if}
				</div>
			</div>

			<div class="flex items-center justify-end gap-2">
				<SwitchWithIcons
					offIcon={EyeOff}
					onIcon={Eye}
					size="md"
					checked={record.published}
					onCheckedChange={(value) => updateCredentialIssuerPublished(record, value)}
				/>

				<Button
					variant="outline"
					size="icon"
					disabled={!record.imported}
					onclick={() => refreshCredentialIssuer(record.url)}
				>
					<RefreshCwIcon />
				</Button>

				<RecordEdit {record} onSuccess={onEditSuccess}>
					{#snippet button({ triggerAttributes, icon: Icon })}
						<Button variant="outline" size="icon" {...triggerAttributes}>
							<Icon />
						</Button>
					{/snippet}
				</RecordEdit>

				<RecordDelete {record}>
					{#snippet button({ triggerAttributes, icon: Icon })}
						<Button variant="outline" size="icon" {...triggerAttributes}>
							<Icon />
						</Button>
					{/snippet}
				</RecordDelete>
			</div>

			<Separator />

			{#snippet infoLink(props: { label: string; href?: string })}
				<div class="flex items-center gap-1">
					<T class="text-nowrap">{props.label}:</T>
					{#if props.href}
						<A class="link-sm block truncate" target="_blank" href={props.href}>
							{props.href}
						</A>
					{:else}
						<T class="text-gray-400">-</T>
					{/if}
				</div>
			{/snippet}

			<div class="space-y-1 text-xs">
				<div class="text-xs">
					<T class="mb-3 mt-0.5 text-xs text-gray-400">
						{record.description}
					</T>
				</div>

				{@render infoLink({ label: 'URL', href: record.url })}
				{@render infoLink({ label: 'Repository', href: record.repo_url })}
				{@render infoLink({ label: 'Homepage', href: record.homepage_url })}
				{#if record.workflow_url}
					{@render infoLink({ label: 'Import results', href: record.workflow_url })}
				{/if}
			</div>

			<Separator />

			<CollectionManager
				collection="credentials"
				queryOptions={{ filter: `credential_issuer = '${record.id}'` }}
				hide={['empty_state']}
			>
				{#snippet top({ records })}
					<div class="flex items-center justify-between">
						{#if records.length > 0}
							<T class="text-sm font-semibold">
								{m.count_available_credentials({
									number: records.length
								})}
							</T>
						{:else}
							<T class="text-gray-300">{m.No_credentials_available()}</T>
						{/if}
						<RecordCreate>
							{#snippet button({ triggerAttributes, icon })}
								{@render blueButton({
									triggerAttributes,
									icon,
									text: m.Add_new_credential()
								})}
							{/snippet}

							{#snippet form({ closeSheet })}
								<CredentialForm credentialIssuer={record} onSuccess={closeSheet} />
							{/snippet}
						</RecordCreate>
					</div>
				{/snippet}

				{#snippet records({ records: credentials, reloadRecords })}
					<ul class="space-y-2">
						{#each credentials as credential (credential.id)}
							<li
								class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2"
							>
								<LabelLink
									label={credential.display_name || credential.name}
									href="/marketplace/credentials/{organization?.canonified_name}/{credential.canonified_name}"
									published={credential.published && record.published}
									class="min-w-0 flex-1 break-words"
								/>

								<div class="flex items-center gap-1">
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

									<SwitchWithIcons
										offIcon={EyeOff}
										onIcon={Eye}
										size="sm"
										disabled={!record.published}
										checked={credential.published}
										onCheckedChange={(value) =>
											updateCredentialPublished(credential, value)}
									/>

									{#if !credential.imported}
										<RecordClone
											collectionName="credentials"
											record={credential}
											onSuccess={reloadRecords}
										/>
									{/if}

									<EditCredentialDialog
										{credential}
										credentialIssuer={record}
										onSuccess={onEditSuccess}
									/>

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
				{/snippet}
			</CollectionManager>
		</div>
	</Card>
{/snippet}

{#snippet blueButton(props: { triggerAttributes: object; icon: IconComponent; text: string })}
	<Button
		variant="link"
		size="sm"
		class="h-8 gap-1 px-2 text-blue-600 hover:cursor-pointer hover:bg-blue-50 hover:no-underline"
		{...props.triggerAttributes}
	>
		<Icon src={props.icon} />
		{props.text}
	</Button>
{/snippet}

<style lang="postcss">
	:global(.link-sm) {
		@apply cursor-pointer truncate !text-gray-400 underline underline-offset-2;
	}
</style>
