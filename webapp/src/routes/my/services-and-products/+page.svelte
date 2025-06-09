<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { m } from '@/i18n';
	import { Pencil, Plus } from 'lucide-svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { buttonVariants } from '@/components/ui/button';
	import CredentialIssuerForm from './credential-issuer-form.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { String } from 'effect';
	import { Badge } from '@/components/ui/badge';
	import Button from '@/components/ui-custom/button.svelte';
	import EditCredentialDialog from './edit-credential-dialog.svelte';
	import { RecordDelete, RecordEdit } from '@/collections-components/manager';

	import Separator from '@/components/ui/separator/separator.svelte';

	import Sheet from '@/components/ui-custom/sheet.svelte';
	import NewWalletForm from './wallet-form.svelte';
	import {
		Collections,
		type CredentialIssuersResponse,
		type CredentialsResponse,
		type WalletsResponse
	} from '@/pocketbase/types';
	import type { ConformanceCheck } from './wallet-form-checks-table.svelte';
	import A from '@/components/ui-custom/a.svelte';
	import Card from '@/components/ui-custom/plainCard.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import StandardAndVersionField from '$lib/standards/standard-and-version-field.svelte';
	import { CheckboxField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';

	//

	let { data } = $props();
	const organizationId = $derived(data.organization?.id);

	let isCredentialIssuerModalOpen = $state(false);
</script>

<div class="space-y-12">
	<div class="space-y-4">
		<CollectionManager
			collection="credential_issuers"
			queryOptions={{
				expand: ['credentials_via_credential_issuer'],
				filter: `owner.id = '${organizationId}'`
			}}
			editFormFieldsOptions={{ exclude: ['owner', 'url'] }}
			subscribe="expanded_collections"
		>
			{#snippet top({ Header })}
				<Header title={m.Credential_issuers()}>
					{#snippet right()}
						{@render CreateCredentialIssuerModal()}
					{/snippet}
				</Header>
			{/snippet}

			{#snippet records({ records, reloadRecords })}
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					{#each records as record}
						{@const credentials =
							record.expand?.credentials_via_credential_issuer ?? []}
						{@render CredentialIssuerCard({
							credentialIssuer: record,
							credentials,
							onEditSuccess: reloadRecords
						})}
					{/each}
				</div>
			{/snippet}
		</CollectionManager>
	</div>

	<div class="space-y-4">
		<CollectionManager
			collection="wallets"
			queryOptions={{
				filter: `owner.id = '${organizationId}'`
			}}
		>
			{#snippet top({ Header })}
				<Header title="Wallets">
					{#snippet right()}
						{@render NewWalletFormSnippet()}
					{/snippet}
				</Header>
			{/snippet}

			{#snippet records({ records, reloadRecords })}
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					{#each records as record (record.id)}
						{@render WalletCard(record, reloadRecords)}
					{/each}
				</div>
			{/snippet}
		</CollectionManager>
	</div>

	<div class="space-y-4">
		<CollectionManager
			collection="verifiers"
			queryOptions={{
				filter: `owner.id = '${organizationId}'`
			}}
			formFieldsOptions={{
				exclude: ['owner', 'conformance_checks'],
				snippets: {
					standard_and_version,
					published,
					description
				},
				order: ['published']
			}}
		>
			{#snippet top({ Header })}
				<Header title="Verifiers"></Header>
			{/snippet}

			{#snippet records({ records, Card })}
				{#each records as record}
					<Card {record} class="bg-background" hide={['select', 'share']}>
						<T class="font-bold">{record.name}</T>
						<T>{record.url}</T>
					</Card>
				{/each}
			{/snippet}
		</CollectionManager>

		{#snippet standard_and_version({ form }: FieldSnippetOptions<'verifiers'>)}
			<StandardAndVersionField {form} name="standard_and_version" />
		{/snippet}

		{#snippet published({ form }: FieldSnippetOptions<'verifiers'>)}
			<div class="flex justify-end gap-2">
				<CheckboxField {form} name="published" options={{ label: m.Published() }} />
			</div>
		{/snippet}

		{#snippet description({ form }: FieldSnippetOptions<'verifiers'>)}
			<MarkdownField {form} name="description" />
		{/snippet}
	</div>
</div>

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
						<T class="font-bold">
							{#if !record.published}
								{title}
							{:else}
								<A href="/marketplace/{Collections.CredentialIssuers}/{record.id}">
									{title}
								</A>
							{/if}
						</T>
						{#if record.published}
							<Badge variant="default">{m.Published()}</Badge>
						{/if}
					</div>

					<T class="mt-1 truncate text-xs text-gray-400">
						{record.url}
					</T>
				</div>

				<div class="flex items-center gap-1">
					<RecordEdit {record} onSuccess={onEditSuccess}>
						{#snippet button({ triggerAttributes, icon: Icon })}
							<Button variant="outline" size="sm" class="p-2" {...triggerAttributes}>
								<Icon />
							</Button>
						{/snippet}
					</RecordEdit>

					<RecordDelete {record}>
						{#snippet button({ triggerAttributes, icon: Icon })}
							<Button variant="outline" size="sm" class="p-2" {...triggerAttributes}>
								<Icon />
							</Button>
						{/snippet}
					</RecordDelete>
				</div>
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
						<li class="bg-muted flex items-center justify-between rounded-md p-2 px-4">
							<div class="flex items-center gap-2">
								{#if !credential.published}
									{credential.key}
								{:else}
									<A href="/marketplace/credentials/{credential.id}">
										{credential.key}
									</A>
								{/if}

								{#if credential.published}
									<Badge variant="default">{m.Published()}</Badge>
								{/if}
							</div>

							<div class="flex items-center gap-1">
								<EditCredentialDialog {credential} onSuccess={onEditSuccess} />
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</Card>
{/snippet}

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

{#snippet WalletCard(wallet: WalletsResponse, onEditSuccess: () => void)}
	<Card class="bg-background overflow-auto">
		{@const conformanceChecks = wallet.conformance_checks as
			| ConformanceCheck[]
			| null
			| undefined}
		<div class="space-y-4 overflow-scroll">
			<div class="flex flex-row items-start justify-between gap-4">
				<div>
					<div class="flex items-center gap-2">
						<T class="font-bold">
							{#if !wallet.published}
								{wallet.name}
							{:else}
								<A href="/apps/{wallet.id}">{wallet.name}</A>
							{/if}
						</T>
						{#if wallet.published}
							<Badge variant="default">{m.Published()}</Badge>
						{/if}
					</div>
					<T class="mt-1 text-xs text-gray-400">
						<RenderMd content={wallet.description} />
					</T>
				</div>

				<div class="flex items-center gap-1">
					{@render UpdateWalletFormSnippet(wallet.id, wallet, onEditSuccess)}

					<RecordDelete record={wallet}>
						{#snippet button({ triggerAttributes, icon: Icon })}
							<Button variant="outline" size="sm" class="p-2" {...triggerAttributes}>
								<Icon />
							</Button>
						{/snippet}
					</RecordDelete>
				</div>
			</div>

			<Separator />

			<div class="flex flex-wrap gap-2">
				{#if conformanceChecks && conformanceChecks.length > 0}
					{#each conformanceChecks as check}
						<Badge variant={check.status === 'success' ? 'secondary' : 'destructive'}>
							{check.test}
						</Badge>
					{/each}
				{:else}
					<T class="text-gray-300">
						{m.No_conformance_checks_available()}
					</T>
				{/if}
			</div>
		</div>
	</Card>
{/snippet}

{#snippet NewWalletFormSnippet()}
	<Sheet>
		{#snippet trigger({ sheetTriggerAttributes })}
			<Button {...sheetTriggerAttributes}><Plus />Add new wallet</Button>
		{/snippet}

		{#snippet content({ closeSheet })}
			<div class="space-y-6">
				<T tag="h3">Add a new wallet</T>
				<NewWalletForm onSuccess={closeSheet} />
			</div>
		{/snippet}
	</Sheet>
{/snippet}

{#snippet UpdateWalletFormSnippet(
	walletId: string,
	initialData: Partial<WalletsResponse>,
	onEditSuccess: () => void
)}
	<Sheet>
		{#snippet trigger({ sheetTriggerAttributes })}
			<Button variant="outline" size="sm" class="p-2" {...sheetTriggerAttributes}>
				<Pencil />
			</Button>
		{/snippet}

		{#snippet content({ closeSheet })}
			<div class="space-y-6">
				<T tag="h3">Add a new wallet</T>
				<NewWalletForm
					{walletId}
					{initialData}
					onSuccess={() => {
						onEditSuccess();
						closeSheet();
					}}
				/>
			</div>
		{/snippet}
	</Sheet>
{/snippet}
