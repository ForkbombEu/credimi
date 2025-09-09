<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { runWithLoading } from '$lib/utils';
	import { String } from 'effect';
	import { Eye, EyeOff, Plus, RefreshCwIcon } from 'lucide-svelte';

	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import { RecordDelete, RecordEdit } from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { buttonVariants } from '@/components/ui/button';
	import { Card } from '@/components/ui/card';
	import * as Dialog from '@/components/ui/dialog';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { Collections } from '@/pocketbase/types';

	import CredentialIssuerForm from './credential-issuer-form.svelte';
	import EditCredentialDialog from './edit-credential-dialog.svelte';
	import { fetchCredentialIssuer } from './utils';

	//

	type Props = {
		organizationId?: string;
		id?: string;
	};

	let { organizationId, id }: Props = $props();

	let isCredentialIssuerModalOpen = $state(false);

	//

	function updatePublished(
		collection: 'credential_issuers' | 'credentials',
		recordId: string,
		published: boolean,
		onSuccess: () => void
	) {
		pb.collection(collection)
			.update(recordId, {
				published
			})
			.then(() => {
				onSuccess();
			});
	}

	async function refreshCredentialIssuer(url: string) {
		runWithLoading({
			fn: () => fetchCredentialIssuer(url),
			loadingText: 'Updating credential issuer...',
			errorText: 'Failed to refresh credential issuer'
		});
	}
</script>

<CollectionManager
	collection="credential_issuers"
	queryOptions={{
		expand: ['credentials_via_credential_issuer'],
		filter: `owner.id = '${organizationId}'`,
		sort: ['created', 'DESC']
	}}
	editFormFieldsOptions={{ exclude: ['owner', 'url', 'published', 'imported'] }}
	subscribe="expanded_collections"
>
	{#snippet top({ Header })}
		<Header title={m.Credential_issuers()} {id}>
			{#snippet right()}
				{@render CreateCredentialIssuerModal()}
			{/snippet}
		</Header>
	{/snippet}

	{#snippet records({ records, reloadRecords })}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each records as record}
				{@const credentials = record.expand?.credentials_via_credential_issuer ?? []}
				{@render CredentialIssuerCard({
					credentialIssuer: record,
					credentials,
					onEditSuccess: reloadRecords
				})}
			{/each}
		</div>
	{/snippet}
</CollectionManager>

<!--  -->

{#snippet CreateCredentialIssuerModal()}
	<Dialog.Root bind:open={isCredentialIssuerModalOpen}>
		<Dialog.Trigger class={buttonVariants({ variant: 'default' })}>
			<Plus />
			{m.Add_new_credential_issuer()}
		</Dialog.Trigger>

		<Dialog.Content class=" sm:max-w-[425px]">
			<Dialog.Header>
				<Dialog.Title>{m.Add_new_credential_issuer()}</Dialog.Title>
			</Dialog.Header>

			<div class="pt-8">
				<CredentialIssuerForm
					onSuccess={() => {
						isCredentialIssuerModalOpen = false;
					}}
				/>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/snippet}

<!--  -->

{#snippet CredentialIssuerCard(props: {
	credentialIssuer: CredentialIssuersResponse;
	credentials: CredentialsResponse[];
	onEditSuccess: () => void;
})}
	{@const { credentialIssuer: record, credentials, onEditSuccess } = props}
	{@const title = String.isNonEmpty(record.name) ? record.name : '[no_title]'}

	<Card class="!p-4">
		<div class="space-y-4">
			<div class="flex items-start justify-between gap-6">
				<Avatar
					src={record.logo_url}
					class="rounded-sm border"
					fallback={record.name.slice(0, 2)}
				/>

				<div class="w-0 grow">
					<div class="flex items-center gap-2">
						<T class="truncate font-bold">
							{#if !record.published}
								{title}
							{:else}
								<A
									href="/marketplace/{Collections.CredentialIssuers}/{record.id}"
									class="truncate underline underline-offset-2 hover:!no-underline"
								>
									{title}
								</A>
							{/if}
						</T>
						{#if record.imported}
							<Badge variant="secondary">{m.Imported()}</Badge>
						{/if}
					</div>

					<div class="text-xs">
						<T class="mb-3 mt-0.5 text-xs text-gray-400">
							{record.description}
						</T>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<SwitchWithIcons
						offIcon={EyeOff}
						onIcon={Eye}
						size="md"
						checked={record.published}
						onCheckedChange={() =>
							updatePublished(
								'credential_issuers',
								record.id,
								!record.published,
								onEditSuccess
							)}
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
			</div>
			<div class="!mt-1 ml-16 text-xs">
				<div class="flex items-center gap-1">
					<T>URL:</T>
					<A class="link-sm" target="_blank" href={record.url}>
						{record.url}
					</A>
				</div>

				<div class="flex items-center gap-1">
					<T>Repository:</T>
					<A class="link-sm" target="_blank" href={record.repo_url}>
						{record.repo_url}
					</A>
				</div>

				<div class="flex items-center gap-1">
					<T>Homepage:</T>
					<A class="link-sm" target="_blank" href={record.homepage_url}>
						{record.homepage_url}
					</A>
				</div>
				{#if record.workflow_url}
					<div class="flex items-center gap-1">
						<T class="text-nowrap">{m.Import_results() + ':'}</T>
						<A class="link-sm" target="_blank" href={record.workflow_url}>
							{record.workflow_url}
						</A>
					</div>
				{/if}
			</div>

			<Separator />

			{#if credentials.length === 0}
				<T class="text-gray-300">{m.No_credentials_available()}</T>
			{:else}
				<T>
					{m.count_available_credentials({
						number: credentials.length
					})}
				</T>

				<ul class="space-y-2">
					{#each credentials as credential}
						<li
							class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2"
						>
							<div class="min-w-0 flex-1 break-words">
								{#if !credential.published || !record.published}
									{credential.name || credential.key}
								{:else}
									<A
										href="/marketplace/credentials/{credential.id}"
										class="break-words font-medium underline underline-offset-2 hover:!no-underline"
									>
										{credential.name || credential.key}
									</A>
								{/if}
							</div>

							<div class="flex items-center gap-2">
								{#if credential.imported}
									<Badge variant="secondary">{m.Imported()}</Badge>
								{/if}
								<SwitchWithIcons
									offIcon={EyeOff}
									onIcon={Eye}
									size="sm"
									disabled={!record.published}
									checked={credential.published}
									onCheckedChange={() =>
										updatePublished(
											'credentials',
											credential.id,
											!credential.published,
											onEditSuccess
										)}
								/>

								<EditCredentialDialog
									{credential}
									credentialIssuer={record}
									onSuccess={onEditSuccess}
								/>
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</Card>
{/snippet}

<style lang="postcss">
	:global(.link-sm) {
		@apply cursor-pointer truncate !text-gray-400 underline underline-offset-2;
	}
</style>
